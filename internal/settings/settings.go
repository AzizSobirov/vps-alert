// Package settings persists the runtime settings that can be changed from
// the bot (thresholds, backup schedule, ...) as a JSON file.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Settings struct {
	Alerts     Alerts    `json:"alerts"`
	Report     Report    `json:"report"`
	Backup     Backup    `json:"backup"`
	MutedUntil time.Time `json:"muted_until"`
}

type Alerts struct {
	CPUPercent   float64 `json:"cpu_percent"`
	MemPercent   float64 `json:"mem_percent"`
	DiskPercent  float64 `json:"disk_percent"`
	InodePercent float64 `json:"inode_percent"`
	// A threshold must be exceeded for this long before an alert fires.
	DurationMin int `json:"duration_min"`
	// While an alert keeps firing, it is repeated at most this often.
	CooldownMin int `json:"cooldown_min"`
}

type Report struct {
	Enabled  bool   `json:"enabled"`
	Time     string `json:"time"`      // HH:MM, server local time
	LastSent string `json:"last_sent"` // YYYY-MM-DD
}

// Database identifies one database to dump before each backup.
type Database struct {
	Engine    string `json:"engine"`              // postgres or mysql
	Container string `json:"container,omitempty"` // Docker container, empty for the host
	Name      string `json:"name"`
}

func (d Database) Label() string {
	source := d.Engine
	if d.Container != "" {
		source = "docker:" + d.Container
	}
	return source + " · " + d.Name
}

type Backup struct {
	Enabled      bool       `json:"enabled"`
	IntervalDays int        `json:"interval_days"`
	Time         string     `json:"time"`   // HH:MM, server local time
	Target       string     `json:"target"` // local, r2 or both
	Paths        []string   `json:"paths"`
	Databases    []Database `json:"databases"`
	KeepDaily    int        `json:"keep_daily"`
	KeepWeekly   int        `json:"keep_weekly"`
	KeepMonthly  int        `json:"keep_monthly"`
	NextRun      time.Time  `json:"next_run"`
	LastRun      time.Time  `json:"last_run"`
	LastOK       bool       `json:"last_ok"`
}

func Defaults() Settings {
	return Settings{
		Alerts: Alerts{
			CPUPercent:   90,
			MemPercent:   90,
			DiskPercent:  85,
			InodePercent: 85,
			DurationMin:  5,
			CooldownMin:  60,
		},
		Report: Report{Enabled: true, Time: "09:00"},
		Backup: Backup{
			IntervalDays: 1,
			Time:         "03:00",
			Target:       "local",
			KeepDaily:    7,
			KeepWeekly:   4,
			KeepMonthly:  6,
		},
	}
}

type Store struct {
	mu   sync.Mutex
	path string
	s    Settings
}

// Open loads settings from path, or starts from defaults if it does not exist.
func Open(path string) (*Store, error) {
	st := &Store{path: path, s: Defaults()}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return st, st.save()
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &st.s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return st, nil
}

// Get returns a copy of the current settings.
func (st *Store) Get() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s.clone()
}

// Update applies fn and persists the result. If fn returns an error nothing
// is changed.
func (st *Store) Update(fn func(*Settings) error) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	next := st.s.clone()
	if err := fn(&next); err != nil {
		return err
	}
	prev := st.s
	st.s = next
	if err := st.save(); err != nil {
		st.s = prev
		return err
	}
	return nil
}

func (s Settings) clone() Settings {
	s.Backup.Paths = append([]string(nil), s.Backup.Paths...)
	s.Backup.Databases = append([]Database(nil), s.Backup.Databases...)
	return s
}

func (st *Store) save() error {
	data, err := json.MarshalIndent(st.s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(st.path), 0o700); err != nil {
		return err
	}
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, st.path)
}

// ParseClock parses an HH:MM string.
func ParseClock(s string) (hour, min int, err error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, fmt.Errorf("time must be HH:MM, got %q", s)
	}
	return t.Hour(), t.Minute(), nil
}

// At returns the given day at clock time hh:mm in now's location.
func At(day time.Time, clock string) time.Time {
	h, m, _ := ParseClock(clock)
	y, mo, d := day.Date()
	return time.Date(y, mo, d, h, m, 0, 0, day.Location())
}

// NextOccurrence returns the first time at clock that is after now.
func NextOccurrence(now time.Time, clock string) time.Time {
	t := At(now, clock)
	if !t.After(now) {
		t = At(now.AddDate(0, 0, 1), clock)
	}
	return t
}

// AdvanceSchedule moves a due run forward by intervalDays until it is in the
// future, so a server that was down for a week runs once, not seven times.
func AdvanceSchedule(prev, now time.Time, clock string, intervalDays int) time.Time {
	if intervalDays < 1 {
		intervalDays = 1
	}
	next := At(prev.AddDate(0, 0, intervalDays), clock)
	for !next.After(now) {
		next = At(next.AddDate(0, 0, intervalDays), clock)
	}
	return next
}
