// Package monitor watches resource usage, raises threshold alerts and sends
// the daily report.
package monitor

import (
	"context"
	"fmt"
	"html"
	"log"
	"net"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/backup"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// An alert resolves only once the value drops this far below its threshold,
// so a value hovering around the line does not flap.
const hysteresis = 5.0

type Monitor struct {
	Server      string
	Store       *settings.Store
	Notify      *telegram.Notifier
	Interval    time.Duration
	DefaultLang string // LANGUAGE from the env file; the /lang setting wins
	DataDir     string // where the disk usage history for forecasts is kept
	// LOGIN_IP_WHITELIST: logins from these networks count as trusted in the
	// daily report.
	EnvWhitelist []*net.IPNet

	states     map[string]*state
	history    *diskHistory
	lastSample time.Time

	// topProcesses is metrics.TopProcesses, replaceable in tests.
	topProcesses func(ctx context.Context, n int) (byCPU, byMem []metrics.Process, err error)
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
			l := i18n.Pick(cfg.Language, m.DefaultLang)
			muted := now.Before(cfg.MutedUntil)
			msgs, fired := m.evaluate(snap, cfg.Alerts, now, l)
			if m.record(snap, now) {
				msgs = append(msgs, m.forecastAlerts(snap, cfg.Alerts.ForecastDays, now, l, muted)...)
			}
			if len(msgs) > 0 && !muted {
				if top := m.topBlock(ctx, fired, l); top != "" {
					msgs = append(msgs, top)
				}
				m.Notify.Send(ctx, fmt.Sprintf("🖥 <b>%s</b>\n\n%s", html.EscapeString(m.Server), strings.Join(msgs, "\n")))
			}
			m.maybeReport(ctx, snap, now, l)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func readings(s metrics.Snapshot, a settings.Alerts, l i18n.Lang) []reading {
	r := []reading{
		{"cpu", i18n.T(l, "monitor.label.cpu"), s.CPUPercent, a.CPUPercent},
		{"mem", i18n.T(l, "monitor.label.mem"), s.MemPercent, a.MemPercent},
	}
	for _, d := range s.Disks {
		r = append(r,
			reading{"disk:" + d.Mount, i18n.T(l, "monitor.label.disk", d.Mount), d.Percent, a.DiskPercent},
			reading{"inode:" + d.Mount, i18n.T(l, "monitor.label.inode", d.Mount), d.InodePercent, a.InodePercent},
		)
	}
	return r
}

// evaluate updates alert state with a new snapshot and returns the alert
// lines that should be sent now, in l, and the keys of the alerts that just
// fired.
func (m *Monitor) evaluate(s metrics.Snapshot, a settings.Alerts, now time.Time, l i18n.Lang) (msgs, fired []string) {
	if m.states == nil {
		m.states = map[string]*state{}
	}
	duration := time.Duration(a.DurationMin) * time.Minute
	cooldown := time.Duration(a.CooldownMin) * time.Minute

	for _, r := range readings(s, a, l) {
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
				msgs = append(msgs, i18n.T(l, "monitor.fired", label, r.value, r.threshold))
				fired = append(fired, r.key)
			} else if st.firing && now.Sub(st.lastSent) >= cooldown {
				st.lastSent = now
				msgs = append(msgs, i18n.T(l, "monitor.still_high",
					label, r.value, metrics.FormatDuration(now.Sub(st.since))))
			}
		case r.value < r.threshold-hysteresis:
			if st.firing {
				msgs = append(msgs, i18n.T(l, "monitor.resolved", label, r.value))
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
	return msgs, fired
}

// topBlock lists the top processes for a CPU or RAM alert that just fired,
// so the cause is visible without logging in.
func (m *Monitor) topBlock(ctx context.Context, fired []string, l i18n.Lang) string {
	var cpu, ram bool
	for _, k := range fired {
		cpu, ram = cpu || k == "cpu", ram || k == "mem"
	}
	if !cpu && !ram {
		return ""
	}
	top := m.topProcesses
	if top == nil {
		top = metrics.TopProcesses
	}
	byCPU, byMem, err := top(ctx, 3)
	if err != nil {
		log.Printf("monitor: top processes: %v", err)
		return ""
	}
	var b strings.Builder
	if cpu && len(byCPU) > 0 {
		b.WriteString("\n" + i18n.T(l, "monitor.top.cpu"))
		for _, p := range byCPU {
			b.WriteString("\n" + i18n.T(l, "monitor.top.line_cpu", html.EscapeString(p.Name), p.PID, p.CPUPercent))
		}
	}
	if ram && len(byMem) > 0 {
		b.WriteString("\n" + i18n.T(l, "monitor.top.mem"))
		for _, p := range byMem {
			b.WriteString("\n" + i18n.T(l, "monitor.top.line_mem", html.EscapeString(p.Name), p.PID, p.MemPercent, metrics.Bytes(p.RSS)))
		}
	}
	return b.String()
}

func (m *Monitor) maybeReport(ctx context.Context, snap metrics.Snapshot, now time.Time, l i18n.Lang) {
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
	b.WriteString(i18n.T(l, "monitor.report.title") + "\n")
	b.WriteString(snap.Format(l, m.Server))
	b.WriteString("\n")
	bk := cfg.Backup
	switch {
	case bk.LastRun.IsZero():
		b.WriteString(i18n.T(l, "monitor.report.backup_never"))
	case bk.LastOK:
		b.WriteString(i18n.T(l, "monitor.report.backup_ok", bk.LastRun.Format("2006-01-02 15:04")))
	default:
		b.WriteString(i18n.T(l, "monitor.report.backup_failed", bk.LastRun.Format("2006-01-02 15:04")))
	}
	if age, stale := backup.StaleFor(bk, now); stale {
		b.WriteString("\n" + i18n.T(l, "monitor.report.backup_stale", metrics.FormatDuration(age)))
	}
	if bk.Enabled {
		b.WriteString("\n" + i18n.T(l, "monitor.report.next_backup", bk.NextRun.Format("2006-01-02 15:04")))
	}
	if lines := m.forecastReport(snap, now, l); len(lines) > 0 {
		b.WriteString("\n\n" + strings.Join(lines, "\n"))
	}
	if summary := m.dailySummary(ctx, l, now); summary != "" {
		b.WriteString("\n\n" + summary)
	}
	m.Notify.Send(ctx, b.String())
}
