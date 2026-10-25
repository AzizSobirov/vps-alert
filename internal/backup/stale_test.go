package backup

import (
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

func TestStaleCheckTiming(t *testing.T) {
	last := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)
	b := settings.Backup{Enabled: true, IntervalDays: 1, LastSuccess: last}
	var h health

	// Daily schedule: the limit is 1 day + 12 hours.
	if _, _, alert := staleCheck(b, &h, last.Add(35*time.Hour)); alert {
		t.Error("alerted before the limit")
	}
	age, got, alert := staleCheck(b, &h, last.Add(37*time.Hour))
	if !alert || got != last || age != 37*time.Hour {
		t.Fatalf("no alert after the limit: %v %v %v", age, got, alert)
	}
	h.LastAlert = last.Add(37 * time.Hour)
	if _, _, alert := staleCheck(b, &h, last.Add(50*time.Hour)); alert {
		t.Error("repeated within 24 hours")
	}
	if _, _, alert := staleCheck(b, &h, last.Add(61*time.Hour)); !alert {
		t.Error("not repeated after 24 hours")
	}

	// Weekly schedule allows 7 days + 12 hours.
	b.IntervalDays = 7
	h = health{}
	if _, _, alert := staleCheck(b, &h, last.Add(7*24*time.Hour)); alert {
		t.Error("weekly schedule alerted after 7 days")
	}
}

func TestStaleCheckOffAndNeverSucceeded(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	h := health{LastAlert: now}
	if _, _, alert := staleCheck(settings.Backup{Enabled: false}, &h, now); alert || h != (health{}) {
		t.Error("schedule off must clear state and not alert")
	}

	// Never succeeded: measured from the first scheduled run.
	due := now.Add(2 * time.Hour)
	b := settings.Backup{Enabled: true, IntervalDays: 1, NextRun: due}
	if _, _, alert := staleCheck(b, &h, now); alert || !h.FirstDue.Equal(due) {
		t.Fatalf("first due = %v", h.FirstDue)
	}
	b.NextRun = due.Add(24 * time.Hour) // the schedule moved on, first due stays
	if _, _, alert := staleCheck(b, &h, due.Add(11*time.Hour)); alert {
		t.Error("alerted within the grace period")
	}
	if _, last, alert := staleCheck(b, &h, due.Add(13*time.Hour)); !alert || !last.IsZero() {
		t.Error("no alert although no backup ever succeeded")
	}
}

func TestStaleForOldSettings(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	// Settings written before LastSuccess existed.
	b := settings.Backup{Enabled: true, IntervalDays: 1, LastRun: now.Add(-72 * time.Hour), LastOK: true}
	if age, stale := StaleFor(b, now); !stale || age != 72*time.Hour {
		t.Errorf("StaleFor = %v %v", age, stale)
	}
	b.LastOK = false
	if _, stale := StaleFor(b, now); stale {
		t.Error("a failed last run is not a known success")
	}
}

func TestStaleStateSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(dir)
	m.healthMu.Lock()
	m.loadHealth().LastAlert = time.Unix(1000, 0)
	m.saveHealth()
	m.healthMu.Unlock()

	m2 := newTestManager(dir)
	m2.healthMu.Lock()
	defer m2.healthMu.Unlock()
	if got := m2.loadHealth().LastAlert; !got.Equal(time.Unix(1000, 0)) {
		t.Errorf("LastAlert after restart = %v", got)
	}
}

func newTestManager(dataDir string) *Manager {
	return &Manager{Cfg: &config.Config{DataDir: dataDir}}
}
