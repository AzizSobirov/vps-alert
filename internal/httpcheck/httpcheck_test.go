package httpcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

// site serves the status code currently stored in code.
func site(t *testing.T, code *atomic.Int32) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "vps-alert" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(int(code.Load()))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestDownAfterTwoFailuresAndRecovery(t *testing.T) {
	var code atomic.Int32
	code.Store(200)
	u := site(t, &code)
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	w := &Watcher{now: func() time.Time { return now }}
	checks := []settings.HTTPCheck{{URL: u}}
	step := func() []string {
		msgs := w.step(context.Background(), i18n.EN, checks)
		now = now.Add(interval)
		return msgs
	}

	if msgs := step(); len(msgs) != 0 {
		t.Fatalf("healthy: %v", msgs)
	}
	if st, ok := Snapshot(u); !ok || !st.Up {
		t.Fatalf("snapshot after success: %+v %v", st, ok)
	}
	code.Store(502)
	if msgs := step(); len(msgs) != 0 {
		t.Fatalf("one failure must not alert: %v", msgs)
	}
	msgs := step()
	if len(msgs) != 1 || !strings.Contains(msgs[0], "is down: 502 Bad Gateway") {
		t.Fatalf("second failure: %v", msgs)
	}
	if st, _ := Snapshot(u); st.Up || st.DownSince.IsZero() || st.Detail != "502 Bad Gateway" {
		t.Errorf("snapshot while down: %+v", st)
	}
	code.Store(200)
	msgs = step()
	if len(msgs) != 1 || !strings.Contains(msgs[0], "up again (down 2m)") {
		t.Fatalf("recovery: %v", msgs)
	}
}

func TestRefusedAndTranslated(t *testing.T) {
	// A port nothing listens on.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	u := "http://" + ln.Addr().String()
	ln.Close()

	w := &Watcher{}
	checks := []settings.HTTPCheck{{URL: u}}
	w.step(context.Background(), i18n.RU, checks)
	msgs := w.step(context.Background(), i18n.RU, checks)
	if len(msgs) != 1 || !strings.Contains(msgs[0], "соединение отклонено") {
		t.Fatalf("got %v", msgs)
	}
}

func TestSlowWarningOnce(t *testing.T) {
	r := result{latency: 6 * time.Second}
	tr := &tracker{}
	var all []string
	for i := 0; i < 5; i++ {
		all = append(all, tr.update(i18n.EN, "https://x", r, time.Now())...)
	}
	if len(all) != 1 || !strings.Contains(all[0], "is slow") {
		t.Fatalf("got %v", all)
	}
	// Fast again resets, so a new slow streak warns again.
	tr.update(i18n.EN, "https://x", result{latency: time.Second}, time.Now())
	all = nil
	for i := 0; i < 3; i++ {
		all = append(all, tr.update(i18n.EN, "https://x", r, time.Now())...)
	}
	if len(all) != 1 {
		t.Fatalf("after recovery: %v", all)
	}
}

func TestRemovedURLForgotten(t *testing.T) {
	var code atomic.Int32
	code.Store(200)
	u := site(t, &code)
	w := &Watcher{}
	w.step(context.Background(), i18n.EN, []settings.HTTPCheck{{URL: u}})
	w.step(context.Background(), i18n.EN, nil)
	if _, ok := Snapshot(u); ok || len(w.trackers) != 0 {
		t.Error("removed URL still tracked")
	}
}

func TestValidate(t *testing.T) {
	for raw, ok := range map[string]bool{
		"https://example.com": true, "http://1.2.3.4:8080/health": true,
		"example.com": false, "ftp://example.com": false, "https://": false,
	} {
		if (Validate(raw) == nil) != ok {
			t.Errorf("Validate(%q): want ok=%v", raw, ok)
		}
	}
}
