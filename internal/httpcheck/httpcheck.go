// Package httpcheck requests configured URLs every minute and alerts when a
// site goes down, comes back or stays slow.
package httpcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

const (
	interval    = time.Minute
	timeout     = 10 * time.Second
	failChecks  = 2               // failures in a row before a down alert
	slowChecks  = 3               // slow responses in a row before a warning
	slowLatency = 5 * time.Second // a response slower than this is slow
	MaxChecks   = 20
)

// Status is the latest result for one URL, for the bot to display.
type Status struct {
	Checked   time.Time
	Up        bool
	Kind      string // why it is down: see Problem
	Detail    string // HTTP status or error text
	Latency   time.Duration
	DownSince time.Time // set while a down alert is active
}

var registry = struct {
	sync.Mutex
	m map[string]Status
}{m: map[string]Status{}}

// Snapshot returns the latest status of url, if it has been checked.
func Snapshot(url string) (Status, bool) {
	registry.Lock()
	defer registry.Unlock()
	s, ok := registry.m[url]
	return s, ok
}

func setStatus(url string, s Status) {
	registry.Lock()
	registry.m[url] = s
	registry.Unlock()
}

func dropStatus(url string) {
	registry.Lock()
	delete(registry.m, url)
	registry.Unlock()
}

// Validate checks that raw is an absolute http or https URL.
func Validate(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return i18n.Err("http.err.url", raw)
	}
	return nil
}

type Watcher struct {
	Cfg    *config.Config
	Store  *settings.Store
	Notify *telegram.Notifier

	client   *http.Client
	trackers map[string]*tracker
	now      func() time.Time
}

type tracker struct {
	fails    int
	down     bool
	since    time.Time // first failure
	slow     int
	slowSent bool
}

type result struct {
	code    int
	reason  string // empty when up
	kind    string // i18n reason key suffix: status, timeout, dns, refused, tls, error
	latency time.Duration
}

func (w *Watcher) Run(ctx context.Context) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		s := w.Store.Get()
		l := i18n.Pick(s.Language, w.Cfg.Language)
		if msgs := w.step(ctx, l, s.HTTPChecks); len(msgs) > 0 && !time.Now().Before(s.MutedUntil) {
			w.Notify.Send(ctx, fmt.Sprintf("🖥 <b>%s</b>\n\n%s", html.EscapeString(w.Cfg.ServerName), strings.Join(msgs, "\n")))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// step checks every URL once, concurrently, and returns the alert lines.
func (w *Watcher) step(ctx context.Context, l i18n.Lang, checks []settings.HTTPCheck) []string {
	if w.client == nil {
		w.client = &http.Client{Timeout: timeout}
	}
	if w.trackers == nil {
		w.trackers = map[string]*tracker{}
	}
	now := time.Now
	if w.now != nil {
		now = w.now
	}

	results := make([]result, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = probe(ctx, w.client, c.URL)
		}()
	}
	wg.Wait()

	var msgs []string
	seen := map[string]bool{}
	for i, c := range checks {
		seen[c.URL] = true
		tr := w.trackers[c.URL]
		if tr == nil {
			tr = &tracker{}
			w.trackers[c.URL] = tr
		}
		msgs = append(msgs, tr.update(l, c.URL, results[i], now())...)
	}
	for u := range w.trackers {
		if !seen[u] {
			delete(w.trackers, u)
			dropStatus(u)
		}
	}
	return msgs
}

func (tr *tracker) update(l i18n.Lang, u string, r result, now time.Time) []string {
	name := html.EscapeString(u)
	var msgs []string
	if r.reason != "" {
		if tr.fails == 0 {
			tr.since = now
		}
		tr.fails++
		if tr.fails >= failChecks && !tr.down {
			tr.down = true
			msgs = append(msgs, i18n.T(l, "http.down", name, html.EscapeString(Problem(l, r.kind, r.reason))))
		}
		tr.slow, tr.slowSent = 0, false
	} else {
		if tr.down {
			msgs = append(msgs, i18n.T(l, "http.up", name, metrics.FormatDuration(now.Sub(tr.since))))
		}
		tr.fails, tr.down = 0, false
		if r.latency > slowLatency {
			tr.slow++
			if tr.slow >= slowChecks && !tr.slowSent {
				tr.slowSent = true
				msgs = append(msgs, i18n.T(l, "http.slow", name, r.latency.Round(100*time.Millisecond).String()))
			}
		} else {
			tr.slow, tr.slowSent = 0, false
		}
	}

	st := Status{Checked: now, Up: r.reason == "", Latency: r.latency}
	if !st.Up {
		st.Kind, st.Detail = r.kind, r.reason
	}
	if tr.down {
		st.DownSince = tr.since
	}
	setStatus(u, st)
	return msgs
}

// Problem renders a failure reason in l. kind is one of status, timeout,
// dns, refused, tls or error; detail is the HTTP status or error text.
func Problem(l i18n.Lang, kind, detail string) string {
	if kind == "status" || kind == "error" {
		return detail
	}
	return i18n.T(l, "http.reason."+kind)
}

func probe(ctx context.Context, client *http.Client, u string) result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return result{reason: err.Error(), kind: "error"}
	}
	req.Header.Set("User-Agent", "vps-alert")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		kind, reason := classify(err)
		return result{reason: reason, kind: kind, latency: time.Since(start)}
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
	r := result{code: resp.StatusCode, latency: time.Since(start)}
	if resp.StatusCode >= 400 {
		r.reason, r.kind = resp.Status, "status"
	}
	return r
}

func classify(err error) (kind, reason string) {
	var dnsErr *net.DNSError
	var certErr *tls.CertificateVerificationError
	var unknownAuth x509.UnknownAuthorityError
	var hostErr x509.HostnameError
	var invalid x509.CertificateInvalidError
	var netErr net.Error
	switch {
	case errors.As(err, &dnsErr):
		return "dns", err.Error()
	case errors.As(err, &certErr), errors.As(err, &unknownAuth), errors.As(err, &hostErr), errors.As(err, &invalid):
		return "tls", err.Error()
	case errors.Is(err, syscall.ECONNREFUSED):
		return "refused", err.Error()
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return "timeout", err.Error()
	}
	// Drop the `Get "https://...":` prefix; the URL is shown anyway.
	var ue *url.Error
	if errors.As(err, &ue) {
		return "error", ue.Err.Error()
	}
	return "error", err.Error()
}
