// Package events counts what the watchers reported (a service stopping, a
// website going down, a file changing, a port opening), so the daily report
// can say how often it happened. Counts survive restarts and are kept for
// a week.
package events

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ServiceDown = "service_down"
	SiteDown    = "site_down"
	FileChange  = "file_change"
	NewPort     = "new_port"
)

const keep = 7 * 24 * time.Hour

var (
	mu  sync.Mutex
	dir string
)

// SetDir enables recording into dir. Until it is called, Add does nothing,
// which keeps tests and the PAM hook from writing files.
func SetDir(d string) {
	mu.Lock()
	dir = d
	mu.Unlock()
}

// Add records n occurrences of kind now.
func Add(kind string, n int) {
	if n <= 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if dir == "" {
		return
	}
	now := time.Now()
	all := load()
	for i := 0; i < n; i++ {
		all[kind] = append(all[kind], now)
	}
	for k, ts := range all {
		all[k] = prune(ts, now.Add(-keep))
	}
	data, err := json.Marshal(all)
	if err == nil {
		tmp := path() + ".tmp"
		if err = os.WriteFile(tmp, data, 0o600); err == nil {
			err = os.Rename(tmp, path())
		}
	}
	if err != nil {
		log.Printf("events: %v", err)
	}
}

// Count returns how many times kind was recorded at or after since.
func Count(kind string, since time.Time) int {
	mu.Lock()
	defer mu.Unlock()
	if dir == "" {
		return 0
	}
	return len(prune(load()[kind], since))
}

func path() string { return filepath.Join(dir, "events.json") }

func load() map[string][]time.Time {
	all := map[string][]time.Time{}
	if data, err := os.ReadFile(path()); err == nil {
		json.Unmarshal(data, &all)
	}
	return all
}

func prune(ts []time.Time, since time.Time) []time.Time {
	var out []time.Time
	for _, t := range ts {
		if !t.Before(since) {
			out = append(out, t)
		}
	}
	return out
}
