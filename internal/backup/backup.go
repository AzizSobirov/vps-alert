// Package backup drives restic to back up configured paths to a local
// repository and/or Cloudflare R2, and runs the backup schedule.
package backup

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/disk"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/dbdump"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

var ErrRunning = errors.New("a backup is already running")

// Refuse to write a local backup when less than this much space is free.
const minFreeBytes = 1 << 30

// restic exit codes (restic >= 0.17).
const (
	exitIncomplete = 3  // snapshot created, but some files could not be read
	exitNoRepo     = 10 // repository does not exist
)

type Manager struct {
	Cfg    *config.Config
	Store  *settings.Store
	Notify *telegram.Notifier

	running atomic.Bool
}

type Result struct {
	Target     string
	SnapshotID string
	FilesNew   int
	DataAdded  uint64
	TotalBytes uint64
	Warning    string
	Err        error
}

type Snapshot struct {
	ShortID string    `json:"short_id"`
	Time    time.Time `json:"time"`
	Paths   []string  `json:"paths"`
	Summary *struct {
		TotalBytesProcessed uint64 `json:"total_bytes_processed"`
	} `json:"summary"`
}

// Available reports why backups cannot run, or nil if they can.
func (m *Manager) Available() error {
	if _, err := exec.LookPath(m.Cfg.ResticBin); err != nil {
		return errors.New("restic is not installed (apt install restic)")
	}
	if m.Cfg.ResticPassword == "" {
		return errors.New("RESTIC_PASSWORD is not set in the env file")
	}
	return nil
}

func (m *Manager) Running() bool { return m.running.Load() }

func Targets(target string) []string {
	switch target {
	case "both":
		return []string{"local", "r2"}
	case "r2":
		return []string{"r2"}
	default:
		return []string{"local"}
	}
}

// Run performs one backup to every configured target and reports the
// outcome to the chat. Only one backup runs at a time.
func (m *Manager) Run(ctx context.Context, reason string) error {
	if !m.running.CompareAndSwap(false, true) {
		return ErrRunning
	}
	defer m.running.Store(false)

	start := time.Now()
	cfg := m.Store.Get().Backup
	dumps, results, err := m.run(ctx, cfg)

	ok := err == nil
	for _, d := range dumps {
		ok = ok && d.Err == nil
	}
	for _, r := range results {
		ok = ok && r.Err == nil
	}
	if uerr := m.Store.Update(func(s *settings.Settings) error {
		s.Backup.LastRun, s.Backup.LastOK = start, ok
		return nil
	}); uerr != nil {
		log.Printf("backup: save status: %v", uerr)
	}

	m.Notify.Send(ctx, m.format(reason, dumps, results, err, time.Since(start)))
	if !ok {
		return errors.New("backup failed")
	}
	return nil
}

func (m *Manager) run(ctx context.Context, cfg settings.Backup) ([]dbdump.Result, []Result, error) {
	if err := m.Available(); err != nil {
		return nil, nil, err
	}
	if len(cfg.Paths) == 0 && len(cfg.Databases) == 0 {
		return nil, nil, errors.New("nothing to back up: add paths with /paths add /path or databases with /db")
	}
	if m.Cfg.PreBackupCmd != "" {
		hookCtx, cancel := context.WithTimeout(ctx, time.Hour)
		defer cancel()
		out, err := exec.CommandContext(hookCtx, "/bin/sh", "-c", m.Cfg.PreBackupCmd).CombinedOutput()
		if err != nil {
			return nil, nil, fmt.Errorf("pre-backup command: %v: %s", err, tail(out))
		}
	}

	paths := cfg.Paths
	var dumps []dbdump.Result
	if len(cfg.Databases) > 0 {
		dir := m.DumpsDir()
		if err := checkFreeSpace(m.Cfg.DataDir); err != nil {
			return nil, nil, fmt.Errorf("database dumps: %w", err)
		}
		// Dumps only need to live until restic has stored them.
		os.RemoveAll(dir)
		defer os.RemoveAll(dir)
		dumps = dbdump.DumpAll(ctx, dir, cfg.Databases)
		if err := dbdump.WriteManifest(dir, dumps); err != nil {
			return dumps, nil, fmt.Errorf("database manifest: %w", err)
		}
		for _, d := range dumps {
			if d.Err == nil {
				paths = append(paths, dir)
				break
			}
		}
	}
	if len(paths) == 0 {
		return dumps, nil, errors.New("every database dump failed")
	}

	var results []Result
	for _, t := range Targets(cfg.Target) {
		results = append(results, m.backupTarget(ctx, t, cfg, paths))
	}
	return dumps, results, nil
}

func (m *Manager) backupTarget(ctx context.Context, target string, cfg settings.Backup, paths []string) Result {
	res := Result{Target: target}
	if target == "local" {
		if err := checkFreeSpace(m.Cfg.LocalRepo); err != nil {
			res.Err = err
			return res
		}
	}
	if err := m.ensureRepo(ctx, target); err != nil {
		res.Err = err
		return res
	}

	args := []string{"backup", "--json", "--tag", "vps-alert", "--host", m.Cfg.ServerName, "--exclude-caches"}
	out, err := m.restic(ctx, target, append(args, paths...)...)
	if code := exitCode(err); code == exitIncomplete {
		res.Warning = "some files could not be read"
	} else if err != nil {
		res.Err = err
		return res
	}
	parseSummary(out, &res)

	if cfg.KeepDaily+cfg.KeepWeekly+cfg.KeepMonthly > 0 {
		_, err := m.restic(ctx, target, "forget", "--prune", "--tag", "vps-alert", "--host", m.Cfg.ServerName,
			"--keep-daily", strconv.Itoa(cfg.KeepDaily),
			"--keep-weekly", strconv.Itoa(cfg.KeepWeekly),
			"--keep-monthly", strconv.Itoa(cfg.KeepMonthly))
		if err != nil {
			res.Warning = strings.TrimPrefix(res.Warning+"; ", "; ") + "cleaning old snapshots failed: " + err.Error()
		}
	}
	return res
}

// Snapshots lists the most recent snapshots in a target, newest first.
func (m *Manager) Snapshots(ctx context.Context, target string, limit int) ([]Snapshot, error) {
	if err := m.Available(); err != nil {
		return nil, err
	}
	out, err := m.restic(ctx, target, "snapshots", "--json", "--tag", "vps-alert")
	if exitCode(err) == exitNoRepo {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snaps []Snapshot
	if err := json.Unmarshal(out, &snaps); err != nil {
		return nil, fmt.Errorf("parse restic snapshots: %w", err)
	}
	// restic lists oldest first.
	for i, j := 0, len(snaps)-1; i < j; i, j = i+1, j-1 {
		snaps[i], snaps[j] = snaps[j], snaps[i]
	}
	if len(snaps) > limit {
		snaps = snaps[:limit]
	}
	return snaps, nil
}

// Schedule runs scheduled backups until ctx is cancelled.
func (m *Manager) Schedule(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		now := time.Now()
		cfg := m.Store.Get().Backup
		if !cfg.Enabled || cfg.NextRun.IsZero() || now.Before(cfg.NextRun) {
			continue
		}
		// Move the schedule forward before running, so a crash mid-backup
		// does not trigger the same run again on restart.
		if err := m.Store.Update(func(s *settings.Settings) error {
			s.Backup.NextRun = settings.AdvanceSchedule(s.Backup.NextRun, now, s.Backup.Time, s.Backup.IntervalDays)
			return nil
		}); err != nil {
			log.Printf("backup schedule: %v", err)
			continue
		}
		if err := m.Run(ctx, "scheduled"); err != nil && !errors.Is(err, ErrRunning) {
			log.Printf("scheduled backup: %v", err)
		}
	}
}

func (m *Manager) repo(target string) (string, []string, error) {
	switch target {
	case "local":
		return m.Cfg.LocalRepo, nil, nil
	case "r2":
		if !m.Cfg.R2Enabled() {
			return "", nil, errors.New("R2 is not configured (set R2_* in the env file)")
		}
		repo := fmt.Sprintf("s3:https://%s.r2.cloudflarestorage.com/%s/%s",
			m.Cfg.R2AccountID, m.Cfg.R2Bucket, m.Cfg.R2Prefix)
		return repo, []string{
			"AWS_ACCESS_KEY_ID=" + m.Cfg.R2AccessKey,
			"AWS_SECRET_ACCESS_KEY=" + m.Cfg.R2SecretKey,
			"AWS_DEFAULT_REGION=auto",
		}, nil
	}
	return "", nil, fmt.Errorf("unknown target %q", target)
}

// Command returns a restic command for target with the repository
// environment set, for callers that stream its input or output.
func (m *Manager) Command(ctx context.Context, target string, args ...string) (*exec.Cmd, error) {
	repo, extra, err := m.repo(target)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, m.Cfg.ResticBin, args...)
	// A minimal environment: restic must not inherit the bot token.
	cmd.Env = append([]string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"RESTIC_REPOSITORY=" + repo,
		"RESTIC_PASSWORD=" + m.Cfg.ResticPassword,
		"RESTIC_CACHE_DIR=" + filepath.Join(m.Cfg.DataDir, "restic-cache"),
	}, extra...)
	return cmd, nil
}

// DumpsDir is where database dumps are written before each backup.
func (m *Manager) DumpsDir() string {
	return filepath.Join(m.Cfg.DataDir, "dumps")
}

func (m *Manager) restic(ctx context.Context, target string, args ...string) ([]byte, error) {
	cmd, err := m.Command(ctx, target, args...)
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), &resticError{op: args[0], err: err, stderr: tail(stderr.Bytes())}
	}
	return stdout.Bytes(), nil
}

type resticError struct {
	op     string
	err    error
	stderr string
}

func (e *resticError) Error() string {
	if e.stderr == "" {
		return fmt.Sprintf("restic %s: %v", e.op, e.err)
	}
	return fmt.Sprintf("restic %s: %s", e.op, e.stderr)
}

func (e *resticError) Unwrap() error { return e.err }

func (m *Manager) ensureRepo(ctx context.Context, target string) error {
	_, err := m.restic(ctx, target, "cat", "config")
	if err == nil {
		return nil
	}
	var re *resticError
	notFound := exitCode(err) == exitNoRepo ||
		(errors.As(err, &re) && (strings.Contains(re.stderr, "does not exist") || strings.Contains(re.stderr, "Is there a repository")))
	if !notFound {
		return err
	}
	if target == "local" {
		if err := os.MkdirAll(m.Cfg.LocalRepo, 0o700); err != nil {
			return err
		}
	}
	_, err = m.restic(ctx, target, "init")
	return err
}

func parseSummary(out []byte, res *Result) {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var line struct {
			MessageType         string `json:"message_type"`
			SnapshotID          string `json:"snapshot_id"`
			FilesNew            int    `json:"files_new"`
			DataAdded           uint64 `json:"data_added"`
			TotalBytesProcessed uint64 `json:"total_bytes_processed"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil || line.MessageType != "summary" {
			continue
		}
		res.SnapshotID = line.SnapshotID
		if len(res.SnapshotID) > 8 {
			res.SnapshotID = res.SnapshotID[:8]
		}
		res.FilesNew, res.DataAdded, res.TotalBytes = line.FilesNew, line.DataAdded, line.TotalBytesProcessed
	}
}

func (m *Manager) format(reason string, dumps []dbdump.Result, results []Result, err error, took time.Duration) string {
	server := html.EscapeString(m.Cfg.ServerName)
	ok := err == nil
	var b strings.Builder
	if len(dumps) > 0 {
		good := 0
		for _, d := range dumps {
			if d.Err == nil {
				good++
			}
		}
		fmt.Fprintf(&b, "\n🗄 Databases: %d/%d dumped", good, len(dumps))
		for _, d := range dumps {
			if d.Err != nil {
				ok = false
				fmt.Fprintf(&b, "\n❌ %s: <code>%s</code>", html.EscapeString(d.DB.Label()), html.EscapeString(d.Err.Error()))
			} else {
				fmt.Fprintf(&b, "\n✅ %s: %s", html.EscapeString(d.DB.Label()), metrics.Bytes(uint64(d.Size)))
			}
		}
		b.WriteString("\n")
	}
	if err != nil {
		fmt.Fprintf(&b, "\n❌ <code>%s</code>", html.EscapeString(err.Error()))
	}
	for _, r := range results {
		if r.Err != nil {
			ok = false
			fmt.Fprintf(&b, "\n❌ <b>%s</b>: <code>%s</code>", r.Target, html.EscapeString(r.Err.Error()))
			continue
		}
		fmt.Fprintf(&b, "\n✅ <b>%s</b>: snapshot <code>%s</code>, +%s new, %s total, %d new files",
			r.Target, r.SnapshotID, metrics.Bytes(r.DataAdded), metrics.Bytes(r.TotalBytes), r.FilesNew)
		if r.Warning != "" {
			fmt.Fprintf(&b, "\n⚠️ %s", html.EscapeString(r.Warning))
		}
	}
	head := "✅ <b>Backup finished</b>"
	if !ok {
		head = "❌ <b>Backup failed</b>"
	}
	return fmt.Sprintf("%s: %s (%s)\n%s\n\n⏱ %s", head, server, reason, b.String(), took.Round(time.Second))
}

func checkFreeSpace(path string) error {
	// The repo may not exist yet; check the nearest existing parent.
	for p := path; ; p = filepath.Dir(p) {
		if _, err := os.Stat(p); err == nil {
			u, err := disk.Usage(p)
			if err != nil {
				return nil
			}
			if u.Free < minFreeBytes {
				return fmt.Errorf("only %s free on %s", metrics.Bytes(u.Free), p)
			}
			return nil
		}
		if p == filepath.Dir(p) {
			return nil
		}
	}
}

func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 0
}

func tail(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 600 {
		s = "…" + s[len(s)-600:]
	}
	return s
}
