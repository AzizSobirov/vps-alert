package monitor

import (
	"strings"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

func TestEvaluateLifecycle(t *testing.T) {
	m := &Monitor{}
	a := settings.Alerts{CPUPercent: 90, DurationMin: 5, CooldownMin: 60}
	t0 := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	cpu := func(v float64) metrics.Snapshot { return metrics.Snapshot{CPUPercent: v} }

	steps := []struct {
		at   time.Duration
		cpu  float64
		want string // substring of the single expected message, "" for none
	}{
		{0, 95, ""},                          // pending
		{4 * time.Minute, 95, ""},            // not long enough
		{5 * time.Minute, 95, "CPU: <b>95%"}, // fires
		{10 * time.Minute, 88, ""},           // hysteresis band, still firing
		{30 * time.Minute, 95, ""},           // cooldown not over
		{66 * time.Minute, 96, "still high"}, // reminder
		{70 * time.Minute, 50, "back to normal"},
		{71 * time.Minute, 50, ""},
	}
	for _, s := range steps {
		msgs, _ := m.evaluate(cpu(s.cpu), a, t0.Add(s.at), i18n.EN)
		if s.want == "" && len(msgs) != 0 {
			t.Errorf("at %v: unexpected %v", s.at, msgs)
		}
		if s.want != "" && (len(msgs) != 1 || !strings.Contains(msgs[0], s.want)) {
			t.Errorf("at %v: got %v, want %q", s.at, msgs, s.want)
		}
	}
}

func TestEvaluateDipResetsPending(t *testing.T) {
	m := &Monitor{}
	a := settings.Alerts{CPUPercent: 90, DurationMin: 5, CooldownMin: 60}
	t0 := time.Now()
	m.evaluate(metrics.Snapshot{CPUPercent: 95}, a, t0, i18n.EN)
	m.evaluate(metrics.Snapshot{CPUPercent: 87}, a, t0.Add(2*time.Minute), i18n.EN)
	if msgs, _ := m.evaluate(metrics.Snapshot{CPUPercent: 95}, a, t0.Add(6*time.Minute), i18n.EN); len(msgs) != 0 {
		t.Errorf("fired although the value dipped: %v", msgs)
	}
}

func TestEvaluateTranslates(t *testing.T) {
	a := settings.Alerts{DiskPercent: 85, DurationMin: 0, CooldownMin: 60}
	s := metrics.Snapshot{Disks: []metrics.Disk{{Mount: "/", Percent: 95}}}
	for l, want := range map[i18n.Lang]string{i18n.EN: "threshold", i18n.UZ: "chegara", i18n.RU: "Диск /"} {
		msgs, _ := (&Monitor{}).evaluate(s, a, time.Now(), l)
		if len(msgs) != 1 || !strings.Contains(msgs[0], want) {
			t.Errorf("%s: got %v, want %q", l, msgs, want)
		}
	}
}
