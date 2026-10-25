// Package sshwatch reads SSH logins from the system logs: it alerts when
// failed logins cross a threshold, which usually means a brute force attack,
// and when someone logs in with a password instead of a key.
package sshwatch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/knownips"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

const (
	checkEvery      = 5 * time.Minute
	loginCheckEvery = time.Minute
	alertWindow     = time.Hour
	topN            = 5
	// The known address list is seeded from this much log history.
	seedPeriod = 30 * 24 * time.Hour
	// When reading a log file directly, only its tail is parsed so a huge
	// auth.log on a server under attack stays cheap.
	maxTailBytes = 20 << 20
)

// Log files used when journald has no sshd entries.
var logFiles = []string{"/var/log/auth.log", "/var/log/secure"}

type Kind int

const (
	kindFailed        Kind = iota + 1 // Failed password/publickey for an existing user
	kindInvalid                       // Invalid user X from IP
	kindFailedInvalid                 // Failed password for invalid user X: already counted by kindInvalid
	kindClosedPreauth                 // connection dropped during authentication
)

// Event is one authentication failure found in a log line.
type Event struct {
	Time time.Time // zero if the line had no recognizable timestamp
	Kind Kind
	User string
	IP   string
	Port string
}

// Login is one successful SSH login.
type Login struct {
	Time   time.Time
	User   string
	IP     string
	Method string // publickey, password, keyboard-interactive/pam, ...
}

// PasswordLike reports whether the login used a password rather than a key.
func (l Login) PasswordLike() bool {
	return l.Method == "password" || strings.HasPrefix(l.Method, "keyboard-interactive")
}

var (
	acceptedRe      = regexp.MustCompile(`^Accepted (\S+) for (.*?) from (\S+) port (\d+)`)
	sshdRe          = regexp.MustCompile(`\bsshd(?:-session)?\[\d+\]: (.*)$`)
	failedRe        = regexp.MustCompile(`^Failed \S+ for (invalid user )?(.*?) from (\S+) port (\d+)`)
	invalidRe       = regexp.MustCompile(`^Invalid user (.*?) from (\S+)(?: port (\d+))?`)
	closedPreauthRe = regexp.MustCompile(`^(?:Connection closed|Disconnected) (?:by|from) (?:authenticating|invalid) user (.*?) (\S+) port (\d+) \[preauth\]`)
	unixTimeRe      = regexp.MustCompile(`^(\d+)(?:\.(\d+))? `)
	isoTimeRe       = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\S+) `)
	syslogTimeRe    = regexp.MustCompile(`^([A-Z][a-z]{2} [ \d]\d \d{2}:\d{2}:\d{2}) `)
)

// ParseLine extracts a failed login from one log line in journald
// short-unix, ISO 8601 (rsyslog) or classic syslog format. now supplies the
// year for syslog timestamps, which have none.
func ParseLine(line string, now time.Time) (Event, bool) {
	m := sshdRe.FindStringSubmatch(line)
	if m == nil {
		return Event{}, false
	}
	msg := m[1]
	var ev Event
	switch {
	case failedRe.MatchString(msg):
		f := failedRe.FindStringSubmatch(msg)
		ev = Event{Kind: kindFailed, User: f[2], IP: f[3], Port: f[4]}
		if f[1] != "" {
			ev.Kind = kindFailedInvalid
		}
	case invalidRe.MatchString(msg):
		f := invalidRe.FindStringSubmatch(msg)
		ev = Event{Kind: kindInvalid, User: f[1], IP: f[2], Port: f[3]}
	case closedPreauthRe.MatchString(msg):
		f := closedPreauthRe.FindStringSubmatch(msg)
		ev = Event{Kind: kindClosedPreauth, User: f[1], IP: f[2], Port: f[3]}
	default:
		return Event{}, false
	}
	ev.Time = parseTime(line, now)
	return ev, true
}

// ParseAccepted extracts a successful login from one log line.
func ParseAccepted(line string, now time.Time) (Login, bool) {
	m := sshdRe.FindStringSubmatch(line)
	if m == nil {
		return Login{}, false
	}
	f := acceptedRe.FindStringSubmatch(m[1])
	if f == nil {
		return Login{}, false
	}
	return Login{Time: parseTime(line, now), Method: f[1], User: f[2], IP: f[3]}, true
}

func parseTime(line string, now time.Time) time.Time {
	if m := unixTimeRe.FindStringSubmatch(line); m != nil {
		sec, _ := strconv.ParseInt(m[1], 10, 64)
		var nsec int64
		if m[2] != "" {
			frac := (m[2] + "000000000")[:9]
			nsec, _ = strconv.ParseInt(frac, 10, 64)
		}
		return time.Unix(sec, nsec)
	}
	if m := isoTimeRe.FindStringSubmatch(line); m != nil {
		if t, err := time.Parse(time.RFC3339Nano, m[1]); err == nil {
			return t
		}
	}
	if m := syslogTimeRe.FindStringSubmatch(line); m != nil {
		t, err := time.ParseInLocation("Jan _2 15:04:05", m[1], now.Location())
		if err != nil {
			return time.Time{}
		}
		t = t.AddDate(now.Year(), 0, 0)
		if t.After(now.Add(24 * time.Hour)) { // December lines read in January
			t = t.AddDate(-1, 0, 0)
		}
		return t
	}
	return time.Time{}
}

type Count struct {
	Key string
	N   int
}

// Stats summarizes the logins over a period.
type Stats struct {
	Total  int     // failed logins
	IPs    []Count // most attempts first
	Users  []Count
	Logins []Login // successful logins, newest first
}

// Aggregate counts the attempts at or after since. Events must be in log
// order. Each failed password or invalid user counts once; a connection that
// was dropped during authentication counts only if nothing else was logged
// for it, so one attempt is never counted twice.
func Aggregate(events []Event, since time.Time) Stats {
	seen := map[string]bool{} // ip+port of connections already counted
	ips, users := map[string]int{}, map[string]int{}
	var st Stats
	for _, ev := range events {
		if !ev.Time.IsZero() && ev.Time.Before(since) {
			continue
		}
		conn := ev.IP + " " + ev.Port
		switch ev.Kind {
		case kindFailedInvalid:
			continue
		case kindClosedPreauth:
			if seen[conn] {
				continue
			}
		}
		if ev.Port != "" {
			seen[conn] = true
		}
		st.Total++
		ips[ev.IP]++
		if ev.User != "" {
			users[ev.User]++
		}
	}
	st.IPs, st.Users = top(ips), top(users)
	return st
}

func top(m map[string]int) []Count {
	out := make([]Count, 0, len(m))
	for k, n := range m {
		out = append(out, Count{k, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Key < out[j].Key
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}

// Summary counts failed SSH logins and lists the successful ones over the
// last period, from journald when it has sshd entries and from the auth log
// file otherwise.
func Summary(ctx context.Context, period time.Duration) (Stats, error) {
	now := time.Now()
	since := now.Add(-period)
	events, logins, err := readLogs(ctx, since, now)
	if err != nil {
		return Stats{}, err
	}
	st := Aggregate(events, since)
	st.Logins = loginsSince(logins, since)
	return st, nil
}

// loginsSince returns the logins at or after since, newest first.
func loginsSince(logins []Login, since time.Time) []Login {
	var out []Login
	for i := len(logins) - 1; i >= 0; i-- {
		if logins[i].Time.IsZero() || !logins[i].Time.Before(since) {
			out = append(out, logins[i])
		}
	}
	return out
}

func readLogs(ctx context.Context, since, now time.Time) ([]Event, []Login, error) {
	if journalHasSSHD(ctx) {
		out, err := exec.CommandContext(ctx, "journalctl", "-t", "sshd", "-t", "sshd-session",
			"--since=@"+strconv.FormatInt(since.Unix(), 10), "-o", "short-unix", "--no-pager", "-q").Output()
		if err != nil {
			return nil, nil, fmt.Errorf("journalctl: %w", err)
		}
		events, logins := parseAll(bytes.NewReader(out), now)
		return events, logins, nil
	}
	for _, f := range logFiles {
		events, logins, err := readFileTail(f, now)
		if err == nil {
			return events, logins, nil
		}
		if !os.IsNotExist(err) {
			return nil, nil, err
		}
	}
	return nil, nil, nil // no source on this system, e.g. a development machine
}

func journalHasSSHD(ctx context.Context) bool {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return false
	}
	out, err := exec.CommandContext(ctx, "journalctl", "-t", "sshd", "-t", "sshd-session", "-n", "1", "-q", "--no-pager").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

func readFileTail(path string, now time.Time) ([]Event, []Login, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size() > maxTailBytes {
		if _, err := f.Seek(-maxTailBytes, io.SeekEnd); err != nil {
			return nil, nil, err
		}
	}
	events, logins := parseAll(f, now)
	return events, logins, nil
}

func parseAll(r io.Reader, now time.Time) ([]Event, []Login) {
	var events []Event
	var logins []Login
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		if ev, ok := ParseLine(sc.Text(), now); ok {
			events = append(events, ev)
		} else if lg, ok := ParseAccepted(sc.Text(), now); ok {
			logins = append(logins, lg)
		}
	}
	return events, logins
}

// SSHDConfig is the part of the effective sshd configuration that decides
// how exposed the server is.
type SSHDConfig struct {
	PasswordAuth bool
	// PermitRootLogin as sshd reports it: yes, prohibit-password (or its
	// old name without-password), forced-commands-only or no.
	PermitRoot string
}

// ReadSSHDConfig asks sshd for its effective configuration (sshd -T), which
// includes defaults and drop-in files. It needs root.
func ReadSSHDConfig(ctx context.Context) (SSHDConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sshd", "-T").Output()
	if err != nil {
		return SSHDConfig{}, fmt.Errorf("sshd -T: %w", err)
	}
	return parseSSHDConfig(string(out)), nil
}

func parseSSHDConfig(out string) SSHDConfig {
	var c SSHDConfig
	for _, line := range strings.Split(out, "\n") {
		key, val, _ := strings.Cut(strings.TrimSpace(line), " ")
		switch strings.ToLower(key) {
		case "passwordauthentication":
			c.PasswordAuth = val == "yes"
		case "permitrootlogin":
			c.PermitRoot = val
		}
	}
	return c
}

// shouldAlert reports whether an alert is due: the threshold is on and
// reached, and no alert was sent within the last window.
func shouldAlert(st Stats, threshold int, lastAlert, now time.Time) bool {
	return threshold > 0 && st.Total >= threshold && now.Sub(lastAlert) >= alertWindow
}

type Watcher struct {
	Cfg    *config.Config
	Store  *settings.Store
	Notify *telegram.Notifier
}

type state struct {
	LastAlert      time.Time `json:"last_alert"`
	LastLoginCheck time.Time `json:"last_login_check"`
}

func (w *Watcher) statePath() string { return filepath.Join(w.Cfg.DataDir, "sshwatch.json") }

func (w *Watcher) knownPath() string { return filepath.Join(w.Cfg.DataDir, knownips.FileName) }

func (w *Watcher) Run(ctx context.Context) {
	var st state
	if data, err := os.ReadFile(w.statePath()); err == nil {
		json.Unmarshal(data, &st)
	}
	w.seedKnownIPs(ctx)
	if st.LastLoginCheck.IsZero() {
		// Only logins from now on are checked, not the whole history.
		st.LastLoginCheck = time.Now()
		w.save(st)
	}

	failures := time.NewTicker(checkEvery)
	defer failures.Stop()
	logins := time.NewTicker(loginCheckEvery)
	defer logins.Stop()
	w.checkFailures(ctx, &st)
	for {
		select {
		case <-ctx.Done():
			return
		case <-failures.C:
			w.checkFailures(ctx, &st)
		case <-logins.C:
			w.checkLogins(ctx, &st)
		}
	}
}

func (w *Watcher) checkFailures(ctx context.Context, st *state) {
	threshold := w.Store.Get().Security.FailedLoginThreshold
	if threshold <= 0 {
		return
	}
	stats, err := Summary(ctx, alertWindow)
	now := time.Now()
	if err != nil {
		log.Printf("sshwatch: %v", err)
	} else if shouldAlert(stats, threshold, st.LastAlert, now) {
		st.LastAlert = now
		w.save(*st)
		w.Notify.Send(ctx, w.message(stats))
	}
}

// checkLogins looks at the logins since the last check: their addresses are
// remembered as known, and password logins are reported.
func (w *Watcher) checkLogins(ctx context.Context, st *state) {
	now := time.Now()
	_, logins, err := readLogs(ctx, st.LastLoginCheck, now)
	if err != nil {
		log.Printf("sshwatch: %v", err)
		return
	}
	var fresh []Login
	for _, lg := range logins {
		if lg.Time.After(st.LastLoginCheck) {
			fresh = append(fresh, lg)
		}
	}
	st.LastLoginCheck = now
	w.save(*st)

	var addrs []string
	for _, lg := range fresh {
		addrs = append(addrs, lg.IP)
	}
	if err := knownips.Add(w.knownPath(), now, addrs...); err != nil {
		log.Printf("sshwatch: %v", err)
	}
	if !w.Store.Get().Security.PasswordLoginAlert {
		return
	}
	for _, lg := range fresh {
		if lg.PasswordLike() {
			w.Notify.Send(ctx, w.passwordMessage(lg))
		}
	}
}

// seedKnownIPs fills the known address list from the log history the first
// time, so logins from your usual addresses are not all flagged as new.
func (w *Watcher) seedKnownIPs(ctx context.Context) {
	if _, exists := knownips.Load(w.knownPath()); exists {
		return
	}
	now := time.Now()
	_, logins, err := readLogs(ctx, now.Add(-seedPeriod), now)
	if err != nil {
		log.Printf("sshwatch: seeding known addresses: %v", err)
	}
	addrs := []string{}
	for _, lg := range logins {
		addrs = append(addrs, lg.IP)
	}
	// Written even when empty, so the PAM hook knows the list is in use.
	if err := knownips.Add(w.knownPath(), now, append(addrs, "")...); err != nil {
		log.Printf("sshwatch: %v", err)
	}
}

func (w *Watcher) passwordMessage(lg Login) string {
	l := i18n.Pick(w.Store.Get().Language, w.Cfg.Language)
	return i18n.T(l, "ssh.password_alert", html.EscapeString(w.Cfg.ServerName), html.EscapeString(lg.User),
		html.EscapeString(lg.IP), lg.Time.Format("15:04"), i18n.DocURL(l, "security.md"))
}

func (w *Watcher) save(st state) {
	data, _ := json.Marshal(st)
	if err := os.WriteFile(w.statePath(), data, 0o600); err != nil {
		log.Printf("sshwatch: %v", err)
	}
}

func (w *Watcher) message(st Stats) string {
	l := i18n.Pick(w.Store.Get().Language, w.Cfg.Language)
	return i18n.T(l, "ssh.alert", html.EscapeString(w.Cfg.ServerName), FormatCount(st.Total)) + "\n\n" + FormatTop(l, st)
}

// FormatTop renders the top IPs and usernames of st as Telegram HTML.
func FormatTop(l i18n.Lang, st Stats) string {
	var b strings.Builder
	if len(st.IPs) > 0 {
		b.WriteString(i18n.T(l, "ssh.top_ips"))
		for _, c := range st.IPs {
			fmt.Fprintf(&b, "\n  <code>%s</code>: %s", html.EscapeString(c.Key), FormatCount(c.N))
		}
	}
	if len(st.Users) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(i18n.T(l, "ssh.top_users"))
		for _, c := range st.Users {
			fmt.Fprintf(&b, "\n  <code>%s</code>: %s", html.EscapeString(c.Key), FormatCount(c.N))
		}
	}
	return b.String()
}

// FormatCount groups digits by thousands: 1240 -> "1 240".
func FormatCount(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + " " + s[i:]
	}
	return s
}
