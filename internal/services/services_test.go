package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

// stub puts fake commands first (and alone) on PATH.
func stub(t *testing.T, scripts map[string]string) {
	dir := t.TempDir()
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
}

func TestCheckParsesSystemctlAndDocker(t *testing.T) {
	stub(t, map[string]string{
		"systemctl": `case "$2" in
nginx.service) printf 'LoadState=loaded\nActiveState=failed\nNRestarts=4\n' ;;
ok.service) printf 'LoadState=loaded\nActiveState=active\nNRestarts=0\n' ;;
*) printf 'LoadState=not-found\nActiveState=inactive\nNRestarts=0\n' ;;
esac`,
		"docker": `case "$4" in
api) echo 'exited|137|2' ;;
web) echo 'running|0|0' ;;
*) echo "Error: No such object: $4" >&2; exit 1 ;;
esac`,
	})
	ctx := context.Background()
	cases := []struct {
		it      Item
		ok      bool
		missing bool
		status  string
	}{
		{Item{Systemd, "nginx.service"}, false, false, "failed"},
		{Item{Systemd, "ok.service"}, true, false, "active"},
		{Item{Systemd, "gone.service"}, false, true, "missing"},
		{Item{Docker, "api"}, false, false, "exited"},
		{Item{Docker, "web"}, true, false, "running"},
		{Item{Docker, "gone"}, false, true, "missing"},
	}
	for _, c := range cases {
		st := Check(ctx, c.it)
		if st.Err != nil || st.OK != c.ok || st.Missing != c.missing || st.Status != c.status {
			t.Errorf("%v: got %+v", c.it, st)
		}
	}
	if st := Check(ctx, Item{Docker, "api"}); st.ExitCode != 137 || st.Restarts != 2 {
		t.Errorf("docker details: %+v", st)
	}
}

func TestDiscoverFiltersInternals(t *testing.T) {
	stub(t, map[string]string{
		"systemctl": `printf 'nginx.service loaded active running nginx\nsystemd-journald.service loaded active running Journal\ngetty@tty1.service loaded active running Getty\npostgresql@16-main.service loaded active running PG\n'`,
		"docker":    `printf 'api\nredis\n'`,
	})
	got := Discover(context.Background())
	want := []Item{{Systemd, "nginx.service"}, {Systemd, "postgresql@16-main.service"}, {Docker, "api"}, {Docker, "redis"}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestDiscoverWithoutTools(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if got := Discover(context.Background()); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

// fake drives step with scripted states and a controllable clock.
type fake struct {
	now    time.Time
	states map[Item]State
}

func (f *fake) watcher() *Watcher {
	return &Watcher{
		now:   func() time.Time { return f.now },
		check: func(_ context.Context, it Item) State { return f.states[it] },
	}
}

func TestDownAfterTwoChecksAndRecovery(t *testing.T) {
	nginx := Item{Systemd, "nginx.service"}
	f := &fake{now: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC), states: map[Item]State{}}
	w := f.watcher()
	step := func(st State) []string {
		f.states[nginx] = st
		msgs := w.step(context.Background(), i18n.EN, []Item{nginx})
		f.now = f.now.Add(interval)
		return msgs
	}

	if msgs := step(State{Status: "active", OK: true}); len(msgs) != 0 {
		t.Fatalf("healthy: %v", msgs)
	}
	if msgs := step(State{Status: "failed"}); len(msgs) != 0 {
		t.Fatalf("one bad check must not alert: %v", msgs)
	}
	msgs := step(State{Status: "failed"})
	if len(msgs) != 1 || !strings.Contains(msgs[0], "<b>nginx</b> is failed") {
		t.Fatalf("second bad check: %v", msgs)
	}
	if msgs := step(State{Status: "failed"}); len(msgs) != 0 {
		t.Fatalf("alert repeated: %v", msgs)
	}
	msgs = step(State{Status: "active", OK: true})
	if len(msgs) != 1 || !strings.Contains(msgs[0], "running again (down 1m)") {
		t.Fatalf("recovery: %v", msgs)
	}
}

func TestShortRestartIsIgnored(t *testing.T) {
	api := Item{Docker, "api"}
	f := &fake{now: time.Now(), states: map[Item]State{}}
	w := f.watcher()
	for _, st := range []State{{Status: "running", OK: true}, {Status: "restarting"}, {Status: "running", OK: true}} {
		f.states[api] = st
		if msgs := w.step(context.Background(), i18n.EN, []Item{api}); len(msgs) != 0 {
			t.Fatalf("%v: %v", st, msgs)
		}
		f.now = f.now.Add(interval)
	}
}

func TestRestartLoop(t *testing.T) {
	api := Item{Docker, "api"}
	f := &fake{now: time.Now(), states: map[Item]State{}}
	w := f.watcher()
	var all []string
	for restarts := 0; restarts <= 4; restarts++ {
		f.states[api] = State{Status: "running", OK: true, Restarts: restarts}
		all = append(all, w.step(context.Background(), i18n.EN, []Item{api})...)
		f.now = f.now.Add(time.Minute)
	}
	if len(all) != 1 || !strings.Contains(all[0], "keeps restarting: 3 restarts") {
		t.Fatalf("got %v", all)
	}
}

func TestMissingReportedOnce(t *testing.T) {
	gone := Item{Systemd, "old.service"}
	f := &fake{now: time.Now(), states: map[Item]State{gone: {Status: "missing", Missing: true}}}
	w := f.watcher()
	first := w.step(context.Background(), i18n.UZ, []Item{gone})
	second := w.step(context.Background(), i18n.UZ, []Item{gone})
	if len(first) != 1 || !strings.Contains(first[0], "topilmadi") || len(second) != 0 {
		t.Fatalf("first %v, second %v", first, second)
	}
}

func TestSelected(t *testing.T) {
	got := Selected(settings.Services{Units: []string{"nginx.service"}, Containers: []string{"api"}})
	if len(got) != 2 || got[0] != (Item{Systemd, "nginx.service"}) || got[1] != (Item{Docker, "api"}) {
		t.Errorf("got %v", got)
	}
	if got[0].Label() != "nginx" || got[1].Label() != "🐳 api" {
		t.Errorf("labels %q %q", got[0].Label(), got[1].Label())
	}
}
