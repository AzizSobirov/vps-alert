package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

// staleGrace is how late a scheduled backup may be before it counts as
// missing, to allow for long backups and short outages.
const staleGrace = 12 * time.Hour

// staleRepeat is how often the stale alert is repeated while it lasts.
const staleRepeat = 24 * time.Hour

// health is the stale-alert state, kept in a file so a restart neither
// repeats an alert nor forgets one.
type health struct {
	FirstDue  time.Time `json:"first_due"`  // first scheduled run, while no backup has ever succeeded
	LastAlert time.Time `json:"last_alert"` // zero when no stale alert is outstanding
}

// lastSuccess is the time of the last fully successful backup. Settings from
// before LastSuccess existed only know whether the last run succeeded.
func lastSuccess(b settings.Backup) time.Time {
	if !b.LastSuccess.IsZero() {
		return b.LastSuccess
	}
	if b.LastOK {
		return b.LastRun
	}
	return time.Time{}
}

func staleLimit(b settings.Backup) time.Duration {
	return time.Duration(max(b.IntervalDays, 1))*24*time.Hour + staleGrace
}

// StaleFor reports how long there has been no successful backup, and
// whether that is longer than the schedule allows. It is false when the
// schedule is off or no backup ever succeeded (failure reports cover that).
func StaleFor(b settings.Backup, now time.Time) (time.Duration, bool) {
	last := lastSuccess(b)
	if !b.Enabled || last.IsZero() {
		return 0, false
	}
	age := now.Sub(last)
	return age, age > staleLimit(b)
}

// staleCheck updates h and decides whether a stale alert is due now. age is
// how long backups have been missing and last the last success (zero if
// none).
func staleCheck(b settings.Backup, h *health, now time.Time) (age time.Duration, last time.Time, alert bool) {
	if !b.Enabled {
		*h = health{}
		return 0, time.Time{}, false
	}
	var deadline time.Time
	if last = lastSuccess(b); last.IsZero() {
		if h.FirstDue.IsZero() {
			h.FirstDue = b.NextRun
			if h.FirstDue.IsZero() {
				h.FirstDue = now
			}
		}
		deadline, age = h.FirstDue.Add(staleGrace), now.Sub(h.FirstDue)
	} else {
		h.FirstDue = time.Time{}
		deadline, age = last.Add(staleLimit(b)), now.Sub(last)
	}
	if now.Before(deadline) {
		return age, last, false
	}
	return age, last, h.LastAlert.IsZero() || now.Sub(h.LastAlert) >= staleRepeat
}

func (m *Manager) healthPath() string {
	return filepath.Join(m.Cfg.DataDir, "backup-health.json")
}

// loadHealth returns the cached state, reading the file the first time.
// The caller holds healthMu.
func (m *Manager) loadHealth() *health {
	if m.health != nil {
		return m.health
	}
	m.health = &health{}
	data, err := os.ReadFile(m.healthPath())
	if err == nil {
		if err := json.Unmarshal(data, m.health); err != nil {
			log.Printf("backup: ignoring %s: %v", m.healthPath(), err)
			*m.health = health{}
		}
	}
	return m.health
}

// saveHealth writes the state. The caller holds healthMu.
func (m *Manager) saveHealth() {
	data, err := json.Marshal(m.health)
	if err != nil {
		return
	}
	tmp := m.healthPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		log.Printf("backup: %v", err)
		return
	}
	if err := os.Rename(tmp, m.healthPath()); err != nil {
		log.Printf("backup: %v", err)
	}
}

// checkStale alerts when scheduled backups have stopped succeeding.
func (m *Manager) checkStale(ctx context.Context, now time.Time) {
	if m.Running() {
		return // a long backup in progress is not a missing one
	}
	s := m.Store.Get()

	m.healthMu.Lock()
	h := m.loadHealth()
	before := *h
	age, last, alert := staleCheck(s.Backup, h, now)
	if alert && now.Before(s.MutedUntil) {
		alert = false // not marked as sent, so it arrives once the mute ends
	}
	if alert {
		h.LastAlert = now
	}
	if *h != before {
		m.saveHealth()
	}
	m.healthMu.Unlock()

	if !alert {
		return
	}
	l := m.Lang()
	lastText := i18n.T(l, "backup.stale.never")
	if !last.IsZero() {
		lastText = last.Format("2006-01-02 15:04")
	}
	m.Notify.Send(ctx, i18n.T(l, "backup.stale", metrics.FormatDuration(age), html.EscapeString(m.Cfg.ServerName), lastText))
}

// resolveStale clears an outstanding stale alert after a successful backup
// and returns the line to add to its report, or "".
func (m *Manager) resolveStale(l i18n.Lang) string {
	m.healthMu.Lock()
	defer m.healthMu.Unlock()
	h := m.loadHealth()
	if h.LastAlert.IsZero() && h.FirstDue.IsZero() {
		return ""
	}
	alerted := !h.LastAlert.IsZero()
	*h = health{}
	m.saveHealth()
	if !alerted {
		return ""
	}
	return fmt.Sprintf("\n\n%s", i18n.T(l, "backup.stale_resolved"))
}
