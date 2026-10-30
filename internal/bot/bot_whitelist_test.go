package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

func TestTrustButton(t *testing.T) {
	b, api := newMenuBot(t)
	b.handle(context.Background(), telegram.Update{CallbackQuery: &telegram.CallbackQuery{ID: "q",
		From: telegram.User{ID: 7, Username: "aziz"}, Message: &telegram.Message{MessageID: 42, Chat: telegram.Chat{ID: 1}},
		Data: "trust:92.63.204.12"}})

	if wl := b.Store.Get().Security.LoginWhitelist; len(wl) != 1 || wl[0] != "92.63.204.12" {
		t.Fatalf("whitelist = %v", wl)
	}
	calls := api.all()
	if !strings.Contains(calls, "editMessageReplyMarkup") || !strings.Contains(calls, "@aziz") {
		t.Errorf("button not removed or confirmation not sent:\n%s", calls)
	}
}

func TestTrustButtonRejectsStrangers(t *testing.T) {
	b, _ := newMenuBot(t)
	b.handle(context.Background(), telegram.Update{CallbackQuery: &telegram.CallbackQuery{ID: "q",
		From: telegram.User{ID: 99}, Message: &telegram.Message{MessageID: 42, Chat: telegram.Chat{ID: 1}},
		Data: "trust:45.148.10.240"}})
	if wl := b.Store.Get().Security.LoginWhitelist; len(wl) != 0 {
		t.Fatalf("a user outside ALLOWED_USER_IDS changed the whitelist: %v", wl)
	}
}

func TestWhitelistCommand(t *testing.T) {
	b, _ := newMenuBot(t)
	ctx := context.Background()
	l := b.language()

	for _, entry := range []string{"deploy", "10.0.0.7/24", "92.63.204.12"} {
		if err := b.whitelistCommand(ctx, l, []string{"add", entry}); err != nil {
			t.Fatalf("add %s: %v", entry, err)
		}
	}
	if err := b.whitelistCommand(ctx, l, []string{"add", "deploy"}); err == nil {
		t.Error("duplicate accepted")
	}
	if err := b.whitelistCommand(ctx, l, []string{"add", "not valid!"}); err == nil {
		t.Error("invalid entry accepted")
	}
	if err := b.whitelistCommand(ctx, l, []string{"rm", "10.0.0.0/24"}); err != nil {
		t.Errorf("rm: %v", err)
	}
	if wl := b.Store.Get().Security.LoginWhitelist; strings.Join(wl, ",") != "deploy,92.63.204.12" {
		t.Errorf("whitelist = %v", wl)
	}
	if st := b.whitelistState(l); !strings.Contains(st, "deploy") {
		t.Errorf("state: %s", st)
	}
}

func TestTrustButtonInAlertChat(t *testing.T) {
	b, api := newMenuBot(t)
	b.Cfg.AlertChatID = -1009
	b.handle(context.Background(), telegram.Update{CallbackQuery: &telegram.CallbackQuery{ID: "q",
		From: telegram.User{ID: 7, Username: "aziz"}, Message: &telegram.Message{MessageID: 42, Chat: telegram.Chat{ID: -1009, Type: "channel"}},
		Data: "trust:92.63.204.12"}})

	if wl := b.Store.Get().Security.LoginWhitelist; len(wl) != 1 {
		t.Fatalf("whitelist = %v", wl)
	}
	calls := api.all()
	if !strings.Contains(calls, `"chat_id":-1009`) || strings.Contains(calls, "leaveChat") {
		t.Errorf("not answered in the alert chat, or left it:\n%s", calls)
	}
}
