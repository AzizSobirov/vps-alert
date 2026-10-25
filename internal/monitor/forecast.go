package monitor

import (
	"encoding/json"
	"html"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
)

const (
	sampleEvery   = 10 * time.Minute
	keepSamples   = 7 * 24 * time.Hour
	fitWindow     = 72 * time.Hour // recent growth predicts better than last week's
	minFitSpan    = 6 * time.Hour
	minFitSamples = 6
	// Growth below this is noise (logs rotating, caches), not a trend.
	minGrowthPerDay = 1 << 20
	// Report lines show forecasts up to this far out; alerts use ForecastDays.
	reportHorizon = 30 * 24 * time.Hour
)

type diskSample struct {
	T    int64  `json:"t"`    // unix seconds
	Used uint64 `json:"used"` // bytes
	Cap  uint64 `json:"cap"`  // used + free: the point where the disk is full
}

// diskHistory is persisted so forecasts survive restarts.
type diskHistory struct {
	Samples map[string][]diskSample `json:"samples"` // by mountpoint
	Alerted map[string]time.Time    `json:"alerted"` // last forecast alert per mountpoint
}

type forecast struct {
	untilFull time.Duration
	perDay    float64 // bytes per day
}

func (m *Monitor) historyPath() string {
	if m.DataDir == "" {
		return ""
	}
	return filepath.Join(m.DataDir, "disk-history.json")
}

// loadHistory reads the history file; a missing or broken file starts empty.
func (m *Monitor) loadHistory() {
	m.history = &diskHistory{Samples: map[string][]diskSample{}, Alerted: map[string]time.Time{}}
	path := m.historyPath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("monitor: %v", err)
		}
		return
	}
	var h diskHistory
	if err := json.Unmarshal(data, &h); err != nil {
		log.Printf("monitor: ignoring %s: %v", path, err)
		return
	}
	if h.Samples != nil {
		m.history.Samples = h.Samples
	}
	if h.Alerted != nil {
		m.history.Alerted = h.Alerted
	}
}

func (m *Monitor) saveHistory() {
	path := m.historyPath()
	if path == "" {
		return
	}
	data, err := json.Marshal(m.history)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		log.Printf("monitor: %v", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("monitor: %v", err)
	}
}

// record adds a sample per disk at most every sampleEvery and drops samples
// older than keepSamples. It reports whether a sample was taken.
func (m *Monitor) record(s metrics.Snapshot, now time.Time) bool {
	if m.history == nil {
		m.loadHistory()
	}
	if !m.lastSample.IsZero() && now.Sub(m.lastSample) < sampleEvery {
		return false
	}
	m.lastSample = now
	cutoff := now.Add(-keepSamples).Unix()
	seen := map[string]bool{}
	for _, d := range s.Disks {
		seen[d.Mount] = true
		samples := append(m.history.Samples[d.Mount], diskSample{T: now.Unix(), Used: d.Used, Cap: d.Used + d.Free})
		i := sort.Search(len(samples), func(i int) bool { return samples[i].T >= cutoff })
		m.history.Samples[d.Mount] = samples[i:]
	}
	// Forget disks that are gone, e.g. an unmounted volume.
	for mount := range m.history.Samples {
		if !seen[mount] {
			delete(m.history.Samples, mount)
			delete(m.history.Alerted, mount)
		}
	}
	m.saveHistory()
	return true
}

// predict fits a line through the recent samples and returns when the disk
// will be full, if usage is clearly growing.
func predict(samples []diskSample, now time.Time) (forecast, bool) {
	from := now.Add(-fitWindow).Unix()
	i := sort.Search(len(samples), func(i int) bool { return samples[i].T >= from })
	samples = samples[i:]
	if len(samples) < minFitSamples || time.Duration(samples[len(samples)-1].T-samples[0].T)*time.Second < minFitSpan {
		return forecast{}, false
	}

	// Least squares over (t, used), with t relative to the first sample to
	// keep the numbers small.
	t0 := float64(samples[0].T)
	var n, sumT, sumU, sumTT, sumTU float64
	for _, s := range samples {
		t, u := float64(s.T)-t0, float64(s.Used)
		n++
		sumT += t
		sumU += u
		sumTT += t * t
		sumTU += t * u
	}
	den := n*sumTT - sumT*sumT
	if den == 0 {
		return forecast{}, false
	}
	slope := (n*sumTU - sumT*sumU) / den // bytes per second
	perDay := slope * 86400
	if perDay < minGrowthPerDay {
		return forecast{}, false
	}
	last := samples[len(samples)-1]
	if last.Used >= last.Cap {
		return forecast{untilFull: 0, perDay: perDay}, true
	}
	secs := float64(last.Cap-last.Used) / slope
	return forecast{untilFull: time.Duration(secs * float64(time.Second)), perDay: perDay}, true
}

// forecastAlerts returns an alert line for each disk predicted to be full
// within days, at most once per day per disk. muted alerts are not marked as
// sent, so they still arrive once the mute ends.
func (m *Monitor) forecastAlerts(s metrics.Snapshot, days int, now time.Time, l i18n.Lang, muted bool) []string {
	if days <= 0 || m.history == nil || muted {
		return nil
	}
	var msgs []string
	for _, d := range s.Disks {
		f, ok := predict(m.history.Samples[d.Mount], now)
		if !ok || f.untilFull > time.Duration(days)*24*time.Hour {
			continue
		}
		if last, ok := m.history.Alerted[d.Mount]; ok && now.Sub(last) < 24*time.Hour {
			continue
		}
		m.history.Alerted[d.Mount] = now
		msgs = append(msgs, i18n.T(l, "monitor.forecast.alert",
			html.EscapeString(i18n.T(l, "monitor.label.disk", d.Mount)), d.Percent,
			eta(l, f.untilFull), metrics.Bytes(uint64(f.perDay))))
	}
	if len(msgs) > 0 {
		m.saveHistory()
	}
	return msgs
}

// forecastReport lists disks predicted to be full within reportHorizon.
func (m *Monitor) forecastReport(s metrics.Snapshot, now time.Time, l i18n.Lang) []string {
	if m.history == nil {
		return nil
	}
	var lines []string
	for _, d := range s.Disks {
		f, ok := predict(m.history.Samples[d.Mount], now)
		if !ok || f.untilFull > reportHorizon {
			continue
		}
		lines = append(lines, i18n.T(l, "monitor.report.forecast",
			html.EscapeString(i18n.T(l, "monitor.label.disk", d.Mount)), eta(l, f.untilFull), metrics.Bytes(uint64(f.perDay))))
	}
	return lines
}

// eta renders a time until full: hours when under two days, else days.
func eta(l i18n.Lang, d time.Duration) string {
	if d < 48*time.Hour {
		h := int(d.Hours())
		if h < 1 {
			h = 1
		}
		return i18n.T(l, "monitor.forecast.hours", h)
	}
	return i18n.T(l, "monitor.forecast.days", int(d.Hours()/24))
}
