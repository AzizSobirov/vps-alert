package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// runOnce runs one Run iteration with an alert about to fire and returns how
// many messages reached Telegram.
func runOnce(t *testing.T, muted bool) int32 {
	var sent atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent.Add(1)
		w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	store, err := settings.Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	nginx := Item{Systemd, "nginx.service"}
	store.Update(func(s *settings.Settings) error {
		s.Services.Units = []string{nginx.Name}
		if muted {
			s.MutedUntil = time.Now().Add(time.Hour)
		}
		return nil
	})
	w := &Watcher{
		Cfg:      &config.Config{ServerName: "test"},
		Store:    store,
		Notify:   &telegram.Notifier{TG: telegram.NewWithURL("t", srv.URL), ChatID: 1},
		check:    func(context.Context, Item) State { return State{Status: "failed"} },
		trackers: map[Item]*tracker{nginx: {bad: 1, since: time.Now()}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	w.Run(ctx)
	return sent.Load()
}

func TestRunRespectsMute(t *testing.T) {
	if n := runOnce(t, true); n != 0 {
		t.Errorf("muted: %d messages sent", n)
	}
	if n := runOnce(t, false); n != 1 {
		t.Errorf("not muted: %d messages sent, want 1", n)
	}
}
