package httpcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

func TestRunRespectsMute(t *testing.T) {
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer broken.Close()

	run := func(muted bool) int32 {
		var sent atomic.Int32
		tg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/sendMessage") {
				sent.Add(1)
			}
			w.Write([]byte(`{"ok":true,"result":{}}`))
		}))
		defer tg.Close()

		store, err := settings.Open(filepath.Join(t.TempDir(), "settings.json"))
		if err != nil {
			t.Fatal(err)
		}
		store.Update(func(s *settings.Settings) error {
			s.HTTPChecks = []settings.HTTPCheck{{URL: broken.URL}}
			if muted {
				s.MutedUntil = time.Now().Add(time.Hour)
			}
			return nil
		})
		w := &Watcher{
			Cfg:      &config.Config{ServerName: "test"},
			Store:    store,
			Notify:   &telegram.Notifier{TG: telegram.NewWithURL("t", tg.URL), ChatID: 1},
			trackers: map[string]*tracker{broken.URL: {fails: 1, since: time.Now()}},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		w.Run(ctx)
		return sent.Load()
	}

	if n := run(true); n != 0 {
		t.Errorf("muted: %d messages sent", n)
	}
	if n := run(false); n != 1 {
		t.Errorf("not muted: %d messages sent, want 1", n)
	}
}
