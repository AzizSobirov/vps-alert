// Package services watches systemd units and Docker containers and alerts
// when one stops, keeps restarting or disappears.
package services

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/events"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

const (
	Systemd = "systemd"
	Docker  = "docker"
)

const (
	interval = 30 * time.Second
	// A bad state must be seen this many checks in a row before alerting, so
	// a normal restart does not page anyone.
	badChecks = 2
	// This many restarts within loopWindow count as a restart loop.
	loopRestarts = 3
	loopWindow   = 10 * time.Minute
)

// Item is one watched unit or container.
type Item struct {
	Kind string // Systemd or Docker
	Name string // unit name as given to systemctl, or container name
}

// Label is a short display name: "nginx" for nginx.service, "🐳 api" for a
// container.
func (i Item) Label() string {
	if i.Kind == Docker {
		return "🐳 " + i.Name
	}
	return strings.TrimSuffix(i.Name, ".service")
}

// Selected returns the items chosen in settings, units first.
func Selected(s settings.Services) []Item {
	var items []Item
	for _, u := range s.Units {
		items = append(items, Item{Systemd, u})
	}
	for _, c := range s.Containers {
		items = append(items, Item{Docker, c})
	}
	return items
}

// State is the result of one check.
type State struct {
	Status   string // systemd ActiveState or Docker State.Status; "missing" if it does not exist
	OK       bool
	Missing  bool
	ExitCode int // Docker only
	Restarts int // systemd NRestarts or Docker RestartCount
	Err      error
}

// Check queries the current state of an item.
func Check(ctx context.Context, it Item) State {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if it.Kind == Docker {
		return checkContainer(ctx, it.Name)
	}
	return checkUnit(ctx, it.Name)
}

func checkUnit(ctx context.Context, name string) State {
	out, err := run(ctx, "systemctl", "show", name, "--property=LoadState,ActiveState,NRestarts")
	if err != nil {
		return State{Status: "unknown", Err: err}
	}
	props := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(l), "="); ok {
			props[k] = v
		}
	}
	if props["LoadState"] == "not-found" {
		return State{Status: "missing", Missing: true}
	}
	st := State{Status: props["ActiveState"]}
	st.Restarts, _ = strconv.Atoi(props["NRestarts"])
	st.OK = st.Status == "active" || st.Status == "reloading"
	return st
}

func checkContainer(ctx context.Context, name string) State {
	out, err := run(ctx, "docker", "inspect", "--format", "{{.State.Status}}|{{.State.ExitCode}}|{{.RestartCount}}", name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such") {
			return State{Status: "missing", Missing: true}
		}
		return State{Status: "unknown", Err: err}
	}
	parts := strings.Split(strings.TrimSpace(out), "|")
	if len(parts) != 3 {
		return State{Status: "unknown", Err: fmt.Errorf("unexpected docker inspect output %q", out)}
	}
	st := State{Status: parts[0]}
	st.ExitCode, _ = strconv.Atoi(parts[1])
	st.Restarts, _ = strconv.Atoi(parts[2])
	st.OK = st.Status == "running"
	return st
}

// OS services that are always running and not worth offering.
var internalPrefixes = []string{
	"systemd-", "getty@", "serial-getty@", "user@", "dbus", "polkit", "ModemManager", "multipathd",
	"udisks2", "packagekit", "snapd", "unattended-upgrades", "networkd-dispatcher", "rsyslog",
	"cron", "atd", "irqbalance", "accounts-daemon", "upower", "thermald", "chrony", "qemu-guest-agent",
	"cloud-", "lvm2-", "iscsid", "open-vm-tools", "vgauth", "wpa_supplicant", "NetworkManager",
	"avahi-daemon", "rtkit-daemon", "fwupd",
}

// Discover lists running services and containers that could be watched.
// Missing systemctl or docker simply yields nothing.
func Discover(ctx context.Context) []Item {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var items []Item
	if out, err := run(ctx, "systemctl", "list-units", "--type=service", "--state=running", "--no-legend", "--plain"); err == nil {
		for _, l := range strings.Split(out, "\n") {
			f := strings.Fields(l)
			if len(f) == 0 || isInternal(f[0]) {
				continue
			}
			items = append(items, Item{Systemd, f[0]})
		}
	}
	if out, err := run(ctx, "docker", "ps", "--format", "{{.Names}}"); err == nil {
		for _, l := range strings.Split(out, "\n") {
			if name := strings.TrimSpace(l); name != "" {
				items = append(items, Item{Docker, name})
			}
		}
	}
	return items
}

func isInternal(unit string) bool {
	for _, p := range internalPrefixes {
		if strings.HasPrefix(unit, p) {
			return true
		}
	}
	return false
}

// StatusText is the localized state, e.g. "failed" or "exited".
func StatusText(l i18n.Lang, status string) string {
	key := "services.state." + status
	if t := i18n.T(l, key); t != key {
		return t
	}
	return status
}

type Watcher struct {
	Cfg    *config.Config
	Store  *settings.Store
	Notify *telegram.Notifier

	trackers map[Item]*tracker
	now      func() time.Time
	check    func(context.Context, Item) State
}

type tracker struct {
	bad      int       // consecutive bad checks
	down     bool      // a down alert was sent
	since    time.Time // first bad check
	missing  bool      // a missing alert was sent
	samples  []sample  // restart counts within loopWindow
	loopSent time.Time
}

type sample struct {
	at       time.Time
	restarts int
}

func (w *Watcher) Run(ctx context.Context) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		s := w.Store.Get()
		l := i18n.Pick(s.Language, w.Cfg.Language)
		if msgs := w.step(ctx, l, Selected(s.Services)); len(msgs) > 0 && !time.Now().Before(s.MutedUntil) {
			w.Notify.Send(ctx, fmt.Sprintf("🖥 <b>%s</b>\n\n%s", html.EscapeString(w.Cfg.ServerName), strings.Join(msgs, "\n")))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// step checks every item once and returns the alert lines to send.
func (w *Watcher) step(ctx context.Context, l i18n.Lang, items []Item) []string {
	if w.trackers == nil {
		w.trackers = map[Item]*tracker{}
	}
	now, check := time.Now, Check
	if w.now != nil {
		now = w.now
	}
	if w.check != nil {
		check = w.check
	}

	var msgs []string
	seen := map[Item]bool{}
	for _, it := range items {
		seen[it] = true
		tr := w.trackers[it]
		if tr == nil {
			tr = &tracker{}
			w.trackers[it] = tr
		}
		st := check(ctx, it)
		if st.Err != nil {
			log.Printf("services: %s: %v", it.Name, st.Err)
			continue // cannot tell; keep the previous state
		}
		msgs = append(msgs, tr.update(l, it, st, now())...)
	}
	for it := range w.trackers {
		if !seen[it] {
			delete(w.trackers, it) // deselected
		}
	}
	return msgs
}

func (tr *tracker) update(l i18n.Lang, it Item, st State, now time.Time) []string {
	name := html.EscapeString(it.Name)
	if it.Kind == Systemd {
		name = html.EscapeString(strings.TrimSuffix(it.Name, ".service"))
	}
	var msgs []string

	if st.Missing {
		if !tr.missing {
			tr.missing = true
			msgs = append(msgs, i18n.T(l, "services.missing."+it.Kind, name))
		}
		return msgs
	}
	tr.missing = false

	if st.OK {
		if tr.down {
			msgs = append(msgs, i18n.T(l, "services.up."+it.Kind, name, metrics.FormatDuration(now.Sub(tr.since))))
		}
		tr.bad, tr.down = 0, false
	} else {
		if tr.bad == 0 {
			tr.since = now
		}
		tr.bad++
		if tr.bad >= badChecks && !tr.down {
			tr.down = true
			events.Add(events.ServiceDown, 1)
			status := html.EscapeString(StatusText(l, st.Status))
			if it.Kind == Docker {
				msgs = append(msgs, i18n.T(l, "services.down.docker", name, status, st.ExitCode))
			} else {
				msgs = append(msgs, i18n.T(l, "services.down.systemd", name, status))
			}
		}
	}

	// Restart loop: restarts counted within the last loopWindow.
	tr.samples = append(tr.samples, sample{now, st.Restarts})
	for len(tr.samples) > 1 && now.Sub(tr.samples[0].at) > loopWindow {
		tr.samples = tr.samples[1:]
	}
	if n := st.Restarts - tr.samples[0].restarts; n >= loopRestarts && now.Sub(tr.loopSent) > loopWindow {
		tr.loopSent = now
		msgs = append(msgs, i18n.T(l, "services.loop."+it.Kind, name, n))
	}
	return msgs
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s", msg)
		}
		return "", err
	}
	return stdout.String(), nil
}
