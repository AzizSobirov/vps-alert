package bot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/backup"
	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// fakeAPI answers every Bot API call with message_id 42 and records calls.
type fakeAPI struct {
	mu    sync.Mutex
	calls []string // "method body"
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.calls = append(f.calls, filepath.Base(r.URL.Path)+" "+string(body))
	f.mu.Unlock()
	w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
}

func (f *fakeAPI) all() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.calls, "\n")
}

func newMenuBot(t *testing.T) (*Bot, *fakeAPI) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	cfg := &config.Config{ChatID: 1, UserIDs: map[int64]bool{7: true}, ServerName: "test", DataDir: t.TempDir()}
	store, err := settings.Open(filepath.Join(cfg.DataDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	b := &Bot{Cfg: cfg, TG: telegram.NewWithURL("token", srv.URL), Store: store,
		Backup: &backup.Manager{Cfg: cfg, Store: store}}
	return b, api
}

func update(text string, replyTo int64, from int64) telegram.Update {
	m := &telegram.Message{MessageID: 50, From: &telegram.User{ID: from}, Chat: telegram.Chat{ID: 1, Type: "group"},
		Date: time.Now().Unix(), Text: text}
	if replyTo != 0 {
		m.ReplyToMessage = &telegram.Message{MessageID: replyTo}
	}
	return telegram.Update{Message: m}
}

func TestMenuPromptAddsPath(t *testing.T) {
	b, api := newMenuBot(t)
	dir := t.TempDir()
	ctx := context.Background()

	b.ask(ctx, "en", "path")
	if b.prompt.msgID != 42 || !strings.Contains(api.all(), "force_reply") {
		t.Fatalf("question not sent as force reply: %+v\n%s", b.prompt, api.all())
	}
	// In a group only a reply to the question counts.
	b.handle(ctx, update(dir, 0, 7))
	if len(b.Store.Get().Backup.Paths) != 0 {
		t.Fatal("a non-reply message in a group was taken as the answer")
	}
	b.handle(ctx, update(dir, 42, 7))
	if p := b.Store.Get().Backup.Paths; len(p) != 1 || p[0] != dir {
		t.Fatalf("paths = %v, want [%s]\n%s", p, dir, api.all())
	}
	if b.prompt.msgID != 0 {
		t.Error("prompt not cleared after the answer")
	}
}

func TestMenuPromptPrivateChat(t *testing.T) {
	b, _ := newMenuBot(t)
	b.Cfg.ChatID, b.Cfg.UserIDs = 7, map[int64]bool{}
	ctx := context.Background()
	private := func(text string) telegram.Update {
		u := update(text, 0, 7)
		u.Message.Chat = telegram.Chat{ID: 7, Type: "private"}
		return u
	}
	dir := t.TempDir()

	b.ask(ctx, "en", "path")
	b.handle(ctx, private("/status")) // a real command is not taken as the answer
	if b.prompt.msgID == 0 {
		t.Fatal("a command cancelled the open question")
	}
	b.handle(ctx, private(dir)) // a path starting with / is
	if p := b.Store.Get().Backup.Paths; len(p) != 1 || p[0] != dir {
		t.Fatalf("paths = %v, want [%s]", p, dir)
	}
}

func TestMenuPromptRejectsStrangersAndExpires(t *testing.T) {
	b, _ := newMenuBot(t)
	ctx := context.Background()

	b.ask(ctx, "en", "url")
	b.handle(ctx, update("https://example.com", 42, 99)) // not an allowed user
	if len(b.Store.Get().HTTPChecks) != 0 {
		t.Fatal("answer from a user outside ALLOWED_USER_IDS accepted")
	}

	b.prompt.at = time.Now().Add(-promptTTL - time.Minute)
	b.handle(ctx, update("https://example.com", 42, 7))
	if len(b.Store.Get().HTTPChecks) != 0 {
		t.Fatal("expired question still accepted an answer")
	}
}

func TestMenuInvalidAnswerShowsError(t *testing.T) {
	b, api := newMenuBot(t)
	ctx := context.Background()
	b.ask(ctx, "en", "url")
	b.handle(ctx, update("not a url", 42, 7))
	if len(b.Store.Get().HTTPChecks) != 0 {
		t.Fatal("invalid URL saved")
	}
	if !strings.Contains(api.all(), "⚠️") {
		t.Error("no error message sent for an invalid answer")
	}
}

func TestMenuButtons(t *testing.T) {
	b, api := newMenuBot(t)
	ctx := context.Background()
	press := func(data string) {
		b.handle(ctx, telegram.Update{CallbackQuery: &telegram.CallbackQuery{ID: "q", From: telegram.User{ID: 7},
			Message: &telegram.Message{MessageID: 42, Chat: telegram.Chat{ID: 1}}, Data: data}})
	}

	press("m:ret:14.8.12")
	if bk := b.Store.Get().Backup; bk.KeepDaily != 14 || bk.KeepWeekly != 8 || bk.KeepMonthly != 12 {
		t.Errorf("retention preset not applied: %+v", bk)
	}
	press("m:tog:files")
	if b.Store.Get().Security.FileWatch {
		t.Error("file watch toggle did not switch off")
	}
	press("m:mute:2h")
	if until := b.Store.Get().MutedUntil; time.Until(until) < 119*time.Minute {
		t.Errorf("mute preset not applied: %v", until)
	}
	press("m:main")
	// Every keyboard sent must keep callback data within Telegram's 64 bytes.
	for _, call := range strings.Split(api.all(), "\n") {
		_, body, _ := strings.Cut(call, " ")
		var req struct {
			ReplyMarkup telegram.Keyboard `json:"reply_markup"`
		}
		json.Unmarshal([]byte(body), &req)
		for _, row := range req.ReplyMarkup.InlineKeyboard {
			for _, bt := range row {
				if len(bt.CallbackData) > 64 {
					t.Errorf("callback data too long: %q", bt.CallbackData)
				}
			}
		}
	}
}
