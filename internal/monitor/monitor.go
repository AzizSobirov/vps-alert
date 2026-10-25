// Package monitor watches resource usage, raises threshold alerts and sends
// the daily report.
package monitor

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// An alert resolves only once the value drops this far below its threshold,
// so a value hovering around the line does not flap.
const hysteresis = 5.0

type Monitor struct {
	Server   string
	Store    *settings.Store
	Notify   *telegram.Notifier
	Interval time.Duration

	states map[string]*state
}

type state struct {
	since    time.Time // first sample above threshold; zero when below
	firing   bool
	lastSent time.Time
}

type reading struct {
	key       string
	label     string
	value     float64
	threshold float64
}

func (m *Monitor) Run(ctx context.Context) {
	if m.Interval == 0 {
		m.Interval = 30 * time.Second
	}
	t := time.NewTicker(m.Interval)
	defer t.Stop()
	for {
		snap, err := metrics.Collect(ctx)
		if err != nil {
			log.Printf("monitor: %v", err)
		} else {
			now := time.Now()
			cfg := m.Store.Get()
			if msgs := m.evaluate(snap, cfg.Alerts, now); len(msgs) > 0 && !now.Before(cfg.MutedUntil) {
				m.Notify.Send(ctx, fmt.Sprintf("🖥 <b>%s</b>\n\n%s", html.EscapeString(m.Server), strings.Join(msgs, "\n")))
			}
			m.maybeReport(ctx, snap, now)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func readings(s metrics.Snapshot, a settings.Alerts) []reading {
	r := []reading{
		{"cpu", "CPU", s.CPUPercent, a.CPUPercent},
		{"mem", "RAM", s.MemPercent, a.MemPercent},
	}
	for _, d := range s.Disks {
		r = append(r,
			reading{"disk:" + d.Mount, "Disk " + d.Mount, d.Percent, a.DiskPercent},
			reading{"inode:" + d.Mount, "Inodes " + d.Mount, d.InodePercent, a.InodePercent},
		)
	}
	return r
}

// evaluate updates alert state with a new snapshot and returns the alert
// lines that should be sent now.
func (m *Monitor) evaluate(s metrics.Snapshot, a settings.Alerts, now time.Time) []string {
	if m.states == nil {
		m.states = map[string]*state{}
	}
	duration := time.Duration(a.DurationMin) * time.Minute
	cooldown := time.Duration(a.CooldownMin) * time.Minute

	var msgs []string
	for _, r := range readings(s, a) {
		if r.threshold <= 0 {
			continue
		}
		st := m.states[r.key]
		if st == nil {
			st = &state{}
			m.states[r.key] = st
		}
		label := html.EscapeString(r.label)

		switch {
		case r.value >= r.threshold:
			if st.since.IsZero() {
				st.since = now
			}
			if !st.firing && now.Sub(st.since) >= duration {
				st.firing, st.lastSent = true, now
				msgs = append(msgs, fmt.Sprintf("🔴 %s: <b>%.0f%%</b> (threshold %.0f%%)", label, r.value, r.threshold))
			} else if st.firing && now.Sub(st.lastSent) >= cooldown {
				st.lastSent = now
				msgs = append(msgs, fmt.Sprintf("🔴 %s still high: <b>%.0f%%</b> for %s",
					label, r.value, metrics.FormatDuration(now.Sub(st.since))))
			}
		case r.value < r.threshold-hysteresis:
			if st.firing {
				msgs = append(msgs, fmt.Sprintf("✅ %s back to normal: %.0f%%", label, r.value))
			}
			*st = state{}
		default:
			// In the hysteresis band: a firing alert stays firing, a pending
			// one starts over because the value was not continuously high.
			if !st.firing {
				st.since = time.Time{}
			}
		}
	}
	return msgs
}

func (m *Monitor) maybeReport(ctx context.Context, snap metrics.Snapshot, now time.Time) {
	cfg := m.Store.Get()
	today := now.Format("2006-01-02")
	if !cfg.Report.Enabled || cfg.Report.LastSent == today || now.Before(settings.At(now, cfg.Report.Time)) {
		return
	}
	if err := m.Store.Update(func(s *settings.Settings) error {
		s.Report.LastSent = today
		return nil
	}); err != nil {
		log.Printf("report: %v", err)
		return
	}

	var b strings.Builder
	b.WriteString("📊 <b>Daily report</b>\n")
	b.WriteString(snap.Format(m.Server))
	b.WriteString("\n")
	bk := cfg.Backup
	switch {
	case bk.LastRun.IsZero():
		b.WriteString("💾 Backup: never run")
	case bk.LastOK:
		fmt.Fprintf(&b, "💾 Last backup: ✅ %s", bk.LastRun.Format("2006-01-02 15:04"))
	default:
		fmt.Fprintf(&b, "💾 Last backup: ❌ failed %s", bk.LastRun.Format("2006-01-02 15:04"))
	}
	if bk.Enabled {
		fmt.Fprintf(&b, "\n⏭ Next backup: %s", bk.NextRun.Format("2006-01-02 15:04"))
	}
	m.Notify.Send(ctx, b.String())
}
