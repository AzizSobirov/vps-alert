// Package filewatch alerts when security-relevant files change (users,
// sudoers, SSH keys and config, cron, systemd units) and when a new port
// starts listening.
package filewatch

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/events"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

const checkEvery = time.Minute

// DefaultPatterns are the watched files. They are globbed at every check, so
// files created later are noticed.
var DefaultPatterns = []string{
	"/etc/passwd",
	"/etc/group",
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/sudoers.d/*",
	"/etc/ssh/sshd_config",
	"/etc/ssh/sshd_config.d/*",
	"/root/.ssh/authorized_keys",
	"/home/*/.ssh/authorized_keys",
	"/etc/crontab",
	"/etc/cron.d/*",
	"/var/spool/cron/crontabs/*",
	"/etc/systemd/system/*.service",
	"/etc/ld.so.preload",
}

// Membership changes in these groups grant admin-like rights.
var privilegedGroups = map[string]bool{"sudo": true, "wheel": true, "admin": true, "docker": true}

// Files bigger than this are compared by metadata only.
const maxRead = 10 << 20

// FileState is what is remembered about a watched file. Entries holds parsed
// lines used to describe a change: passwd user -> uid, group name ->
// members, authorized_keys fingerprint -> "type comment", and for shadow
// user -> a hash of the line, so password hashes are never stored.
type FileState struct {
	SHA256  string            `json:"sha256,omitempty"`
	Size    int64             `json:"size"`
	Mode    uint32            `json:"mode"`
	UID     int               `json:"uid"`
	GID     int               `json:"gid"`
	Entries map[string]string `json:"entries,omitempty"`
}

// Detail is one translated line describing a change.
type Detail struct {
	Key  string
	Args []any
}

type Change struct {
	Path     string
	Kind     string // created, deleted, modified or perms
	Old, New string // permissions and owner, for perms
	Details  []Detail
}

// Scan reads the current state of every file matching patterns. Missing or
// unreadable files are skipped or compared by metadata.
func Scan(patterns []string) map[string]FileState {
	out := map[string]FileState{}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		for _, m := range matches {
			if st, ok := stateOf(m); ok {
				out[m] = st
			}
		}
	}
	return out
}

func stateOf(path string) (FileState, bool) {
	fi, err := os.Lstat(path)
	if err != nil || fi.IsDir() {
		return FileState{}, false
	}
	st := FileState{Size: fi.Size(), Mode: unixMode(fi.Mode())}
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		st.UID, st.GID = int(sys.Uid), int(sys.Gid)
	}
	if !fi.Mode().IsRegular() || fi.Size() > maxRead {
		return st, true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return st, true // e.g. /etc/shadow when not running as root
	}
	sum := sha256.Sum256(data)
	st.SHA256 = hex.EncodeToString(sum[:])
	st.Entries = entries(filepath.Base(path), data)
	return st, true
}

func unixMode(m os.FileMode) uint32 {
	mode := uint32(m.Perm())
	if m&os.ModeSetuid != 0 {
		mode |= 0o4000
	}
	if m&os.ModeSetgid != 0 {
		mode |= 0o2000
	}
	if m&os.ModeSticky != 0 {
		mode |= 0o1000
	}
	return mode
}

func entries(base string, data []byte) map[string]string {
	e := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, ":")
		switch base {
		case "passwd":
			if len(f) >= 3 {
				e[f[0]] = f[2]
			}
		case "group":
			if len(f) >= 4 && privilegedGroups[f[0]] {
				e[f[0]] = normalizeMembers(f[3])
			}
		case "shadow":
			sum := sha256.Sum256([]byte(line))
			e[f[0]] = hex.EncodeToString(sum[:8])
		case "authorized_keys":
			if typ, fp, comment, ok := parseKey(line); ok {
				e[fp] = strings.TrimSpace(typ + " " + comment)
			}
		}
	}
	if len(e) == 0 {
		return nil
	}
	return e
}

func normalizeMembers(s string) string {
	var m []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			m = append(m, x)
		}
	}
	sort.Strings(m)
	return strings.Join(m, ",")
}

// parseKey finds the key in an authorized_keys line, which may start with
// options, and returns its type, OpenSSH SHA256 fingerprint and comment.
func parseKey(line string) (typ, fingerprint, comment string, ok bool) {
	fields := strings.Fields(line)
	for i := 0; i+1 < len(fields); i++ {
		if !isKeyType(fields[i]) {
			continue
		}
		blob, err := base64.StdEncoding.DecodeString(fields[i+1])
		if err != nil {
			continue
		}
		sum := sha256.Sum256(blob)
		return fields[i], "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), strings.Join(fields[i+2:], " "), true
	}
	return "", "", "", false
}

func isKeyType(s string) bool {
	return strings.HasPrefix(s, "ssh-") || strings.HasPrefix(s, "ecdsa-sha2-") ||
		strings.HasPrefix(s, "sk-ssh-") || strings.HasPrefix(s, "sk-ecdsa-")
}

// Diff compares two scans.
func Diff(old, cur map[string]FileState) []Change {
	var changes []Change
	for path, n := range cur {
		o, existed := old[path]
		if !existed {
			c := Change{Path: path, Kind: "created"}
			c.Details = details(filepath.Base(path), nil, n.Entries)
			changes = append(changes, c)
			continue
		}
		if o.SHA256 != n.SHA256 || o.Size != n.Size {
			c := Change{Path: path, Kind: "modified"}
			c.Details = details(filepath.Base(path), o.Entries, n.Entries)
			changes = append(changes, c)
		}
		if o.Mode != n.Mode || o.UID != n.UID || o.GID != n.GID {
			changes = append(changes, Change{Path: path, Kind: "perms", Old: perms(o), New: perms(n)})
		}
	}
	for path := range old {
		if _, ok := cur[path]; !ok {
			changes = append(changes, Change{Path: path, Kind: "deleted"})
		}
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}

func perms(s FileState) string {
	return fmt.Sprintf("%04o %d:%d", s.Mode, s.UID, s.GID)
}

func details(base string, old, cur map[string]string) []Detail {
	var d []Detail
	switch base {
	case "passwd":
		for _, name := range sortedKeys(cur) {
			uid := cur[name]
			prev, existed := old[name]
			switch {
			case uid == "0" && (!existed || prev != "0"):
				d = append(d, Detail{"files.user_root", []any{html.EscapeString(name)}})
			case !existed:
				d = append(d, Detail{"files.user_added", []any{html.EscapeString(name), html.EscapeString(uid)}})
			}
		}
		for _, name := range sortedKeys(old) {
			if _, ok := cur[name]; !ok {
				d = append(d, Detail{"files.user_removed", []any{html.EscapeString(name)}})
			}
		}
	case "group":
		groups := map[string]bool{}
		for g := range old {
			groups[g] = true
		}
		for g := range cur {
			groups[g] = true
		}
		for _, g := range sortedKeys(groups) {
			added, removed := diffMembers(old[g], cur[g])
			for _, u := range added {
				d = append(d, Detail{"files.group_added", []any{html.EscapeString(u), html.EscapeString(g)}})
			}
			for _, u := range removed {
				d = append(d, Detail{"files.group_removed", []any{html.EscapeString(u), html.EscapeString(g)}})
			}
		}
	case "shadow":
		var changed []string
		for _, name := range sortedKeys(cur) {
			if prev, ok := old[name]; ok && prev != cur[name] {
				changed = append(changed, html.EscapeString(name))
			}
		}
		if len(changed) > 0 {
			d = append(d, Detail{"files.shadow_changed", []any{strings.Join(changed, ", ")}})
		}
	case "authorized_keys":
		for _, fp := range sortedKeys(cur) {
			if _, ok := old[fp]; !ok {
				d = append(d, Detail{"files.key_added", []any{html.EscapeString(fp), html.EscapeString(cur[fp])}})
			}
		}
		for _, fp := range sortedKeys(old) {
			if _, ok := cur[fp]; !ok {
				d = append(d, Detail{"files.key_removed", []any{html.EscapeString(fp), html.EscapeString(old[fp])}})
			}
		}
	}
	return d
}

func diffMembers(old, cur string) (added, removed []string) {
	set := func(s string) map[string]bool {
		m := map[string]bool{}
		for _, x := range strings.Split(s, ",") {
			if x != "" {
				m[x] = true
			}
		}
		return m
	}
	o, c := set(old), set(cur)
	for _, u := range sortedKeys(c) {
		if !o[u] {
			added = append(added, u)
		}
	}
	for _, u := range sortedKeys(o) {
		if !c[u] {
			removed = append(removed, u)
		}
	}
	return added, removed
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatChanges renders changes as Telegram HTML lines in l.
func FormatChanges(l i18n.Lang, changes []Change) string {
	var b strings.Builder
	for i, c := range changes {
		if i > 0 {
			b.WriteString("\n")
		}
		path := html.EscapeString(c.Path)
		if c.Kind == "perms" {
			b.WriteString(i18n.T(l, "files.perms", path, c.Old, c.New))
		} else {
			b.WriteString(i18n.T(l, "files."+c.Kind, path))
		}
		for _, d := range c.Details {
			b.WriteString("\n   " + i18n.T(l, d.Key, d.Args...))
		}
	}
	return b.String()
}

// baseline is persisted in DATA_DIR/filewatch.json. A nil map means that
// part was never recorded, so the first check only records.
type baseline struct {
	Files      map[string]FileState `json:"files"`
	Ports      map[string]Port      `json:"ports"`
	LastChange time.Time            `json:"last_change"`
	LastPath   string               `json:"last_path"`
}

func baselinePath(dataDir string) string { return filepath.Join(dataDir, "filewatch.json") }

func loadBaseline(dataDir string) baseline {
	var b baseline
	if data, err := os.ReadFile(baselinePath(dataDir)); err == nil {
		json.Unmarshal(data, &b)
	}
	return b
}

func (b baseline) save(dataDir string) error {
	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	tmp := baselinePath(dataDir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, baselinePath(dataDir))
}

// Status reports how many files are watched and the last recorded change.
func Status(dataDir string) (watched int, lastChange time.Time, lastPath string) {
	b := loadBaseline(dataDir)
	return len(Scan(DefaultPatterns)), b.LastChange, b.LastPath
}

// KnownPorts returns the listening sockets accepted so far.
func KnownPorts(dataDir string) map[string]Port {
	return loadBaseline(dataDir).Ports
}

type Watcher struct {
	Cfg    *config.Config
	Store  *settings.Store
	Notify *telegram.Notifier
}

func (w *Watcher) Run(ctx context.Context) {
	b := loadBaseline(w.Cfg.DataDir)
	ports := &portTracker{known: b.Ports}
	// Start as if both were on, so changes made while vps-alert was stopped
	// are reported. Turning a watch off and on again records a fresh
	// baseline instead of reporting everything that changed meanwhile.
	filesWasOn, portsWasOn := true, true

	t := time.NewTicker(checkEvery)
	defer t.Stop()
	for {
		sec := w.Store.Get().Security
		l := i18n.Pick(w.Store.Get().Language, w.Cfg.Language)
		dirty := false

		if sec.FileWatch {
			cur := Scan(DefaultPatterns)
			if b.Files == nil || !filesWasOn {
				b.Files, dirty = cur, true
			} else if changes := Diff(b.Files, cur); len(changes) > 0 {
				b.Files, b.LastChange, b.LastPath, dirty = cur, time.Now(), changes[0].Path, true
				events.Add(events.FileChange, len(changes))
				w.Notify.Send(ctx, i18n.T(l, "files.alert", html.EscapeString(w.Cfg.ServerName))+"\n\n"+FormatChanges(l, changes))
			}
		}
		filesWasOn = sec.FileWatch

		if sec.PortWatch {
			cur, err := ListPorts(ctx)
			if err != nil {
				log.Printf("filewatch: ports: %v", err)
			} else {
				if ports.known == nil || !portsWasOn {
					ports.reset(cur)
				} else if opened := ports.update(cur); len(opened) > 0 {
					events.Add(events.NewPort, len(opened))
					w.Notify.Send(ctx, i18n.T(l, "ports.alert", html.EscapeString(w.Cfg.ServerName))+"\n\n"+FormatPorts(l, opened, nil))
				}
				if !samePorts(b.Ports, ports.known) {
					b.Ports, dirty = clonePorts(ports.known), true
				}
			}
		}
		portsWasOn = sec.PortWatch

		if dirty {
			if err := b.save(w.Cfg.DataDir); err != nil {
				log.Printf("filewatch: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
