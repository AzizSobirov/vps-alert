package monitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

const gib = 1 << 30

// series returns one sample every 10 minutes for span, growing by perDay
// bytes per day from used, on a disk of capacity cap.
func series(end time.Time, span time.Duration, used, perDay, cap float64) []diskSample {
	var out []diskSample
	for t := end.Add(-span); !t.After(end); t = t.Add(sampleEvery) {
		u := used + perDay*t.Sub(end.Add(-span)).Hours()/24
		out = append(out, diskSample{T: t.Unix(), Used: uint64(u), Cap: uint64(cap)})
	}
	return out
}

func TestPredictGrowing(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	// 70 GiB used of 100, growing 5 GiB/day over 2 days: 80 GiB now, 4 days left.
	f, ok := predict(series(now, 48*time.Hour, 70*gib, 5*gib, 100*gib), now)
	if !ok {
		t.Fatal("no forecast for steady growth")
	}
	if d := f.untilFull.Hours() / 24; d < 3.9 || d > 4.1 {
		t.Errorf("until full = %.2f days, want ~4", d)
	}
	if g := f.perDay / gib; g < 4.9 || g > 5.1 {
		t.Errorf("growth = %.2f GiB/day, want ~5", g)
	}
}

func TestPredictNoTrend(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	cases := map[string][]diskSample{
		"flat":      series(now, 48*time.Hour, 70*gib, 0, 100*gib),
		"shrinking": series(now, 48*time.Hour, 70*gib, -2*gib, 100*gib),
		"too short": series(now, 3*time.Hour, 70*gib, 50*gib, 100*gib),
		"noise":     series(now, 48*time.Hour, 70*gib, 100<<10, 100*gib), // 100 KiB/day
		"empty":     nil,
	}
	for name, samples := range cases {
		if f, ok := predict(samples, now); ok {
			t.Errorf("%s: unexpected forecast %+v", name, f)
		}
	}
}

func TestPredictUsesRecentWindow(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	// A week ago the disk shrank a lot (cleanup); for the last 3 days it grows.
	old := series(now.Add(-4*24*time.Hour), 72*time.Hour, 90*gib, -10*gib, 100*gib)
	recent := series(now, 72*time.Hour, 50*gib, 3*gib, 100*gib)
	if _, ok := predict(append(old, recent...), now); !ok {
		t.Error("old shrinking samples outside the window hid the recent growth")
	}
}

func TestForecastAlertOncePerDayAndPersisted(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	m := &Monitor{DataDir: dir}
	m.loadHistory()
	m.history.Samples["/"] = series(now, 48*time.Hour, 70*gib, 5*gib, 100*gib)
	snap := metrics.Snapshot{Disks: []metrics.Disk{{Mount: "/", Used: 80 * gib, Free: 20 * gib, Percent: 80}}}

	if msgs := m.forecastAlerts(snap, 7, now, i18n.EN, false); len(msgs) != 1 || !strings.Contains(msgs[0], "full in ~4 days") {
		t.Fatalf("first alert: %v", msgs)
	}
	if msgs := m.forecastAlerts(snap, 7, now.Add(time.Hour), i18n.EN, false); len(msgs) != 0 {
		t.Errorf("repeated within a day: %v", msgs)
	}
	if msgs := m.forecastAlerts(snap, 3, now.Add(25*time.Hour), i18n.EN, false); len(msgs) != 0 {
		t.Errorf("alerted although 4 days > 3 day setting: %v", msgs)
	}

	// After a restart the last alert time is remembered.
	m2 := &Monitor{DataDir: dir}
	m2.loadHistory()
	if msgs := m2.forecastAlerts(snap, 7, now.Add(2*time.Hour), i18n.EN, false); len(msgs) != 0 {
		t.Errorf("alert repeated after restart: %v", msgs)
	}
	if msgs := m2.forecastAlerts(snap, 7, now.Add(25*time.Hour), i18n.EN, false); len(msgs) != 1 {
		t.Errorf("no alert the next day: %v", msgs)
	}
}

func TestForecastMutedAndOff(t *testing.T) {
	now := time.Now()
	m := &Monitor{}
	m.loadHistory()
	m.history.Samples["/"] = series(now, 48*time.Hour, 70*gib, 5*gib, 100*gib)
	snap := metrics.Snapshot{Disks: []metrics.Disk{{Mount: "/", Percent: 80}}}
	if msgs := m.forecastAlerts(snap, 7, now, i18n.EN, true); len(msgs) != 0 {
		t.Errorf("alerted while muted: %v", msgs)
	}
	if msgs := m.forecastAlerts(snap, 0, now, i18n.EN, false); len(msgs) != 0 {
		t.Errorf("alerted with forecast off: %v", msgs)
	}
	if msgs := m.forecastAlerts(snap, 7, now, i18n.EN, false); len(msgs) != 1 {
		t.Errorf("muted alert was marked as sent: %v", msgs)
	}
}

func TestRecordSamplesAndCorruptFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "disk-history.json"), []byte("{not json"), 0o600)
	m := &Monitor{DataDir: dir}
	now := time.Now()
	snap := metrics.Snapshot{Disks: []metrics.Disk{{Mount: "/", Used: 10, Free: 90}}}

	if !m.record(snap, now) {
		t.Fatal("first sample not taken")
	}
	if m.record(snap, now.Add(time.Minute)) {
		t.Error("sampled again within 10 minutes")
	}
	m.record(snap, now.Add(11*time.Minute))
	if n := len(m.history.Samples["/"]); n != 2 {
		t.Errorf("samples = %d, want 2", n)
	}
	m.record(metrics.Snapshot{}, now.Add(22*time.Minute))
	if _, ok := m.history.Samples["/"]; ok {
		t.Error("unmounted disk kept")
	}
}

func TestTopBlockOnlyForFiredCPUOrRAM(t *testing.T) {
	m := &Monitor{topProcesses: func(ctx context.Context, n int) ([]metrics.Process, []metrics.Process, error) {
		return []metrics.Process{{PID: 7, Name: "<miner>", CPUPercent: 390}},
			[]metrics.Process{{PID: 8, Name: "java", MemPercent: 61, RSS: 5 * gib}}, nil
	}}
	if got := m.topBlock(context.Background(), []string{"disk:/"}, i18n.EN); got != "" {
		t.Errorf("disk alert got top block: %q", got)
	}
	got := m.topBlock(context.Background(), []string{"cpu"}, i18n.EN)
	if !strings.Contains(got, "&lt;miner&gt;") || strings.Contains(got, "java") {
		t.Errorf("cpu block = %q", got)
	}
	if got := m.topBlock(context.Background(), []string{"mem"}, i18n.EN); !strings.Contains(got, "java") {
		t.Errorf("mem block = %q", got)
	}
}

func TestEvaluateReportsFired(t *testing.T) {
	m := &Monitor{}
	a := settings.Alerts{CPUPercent: 90, MemPercent: 90, DurationMin: 0, CooldownMin: 60}
	_, fired := m.evaluate(metrics.Snapshot{CPUPercent: 95, MemPercent: 10}, a, time.Now(), i18n.EN)
	if len(fired) != 1 || fired[0] != "cpu" {
		t.Errorf("fired = %v", fired)
	}
	_, fired = m.evaluate(metrics.Snapshot{CPUPercent: 95, MemPercent: 10}, a, time.Now().Add(time.Minute), i18n.EN)
	if len(fired) != 0 {
		t.Errorf("still firing reported as fired again: %v", fired)
	}
}
