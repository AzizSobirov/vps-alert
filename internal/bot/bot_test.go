package bot

import (
	"errors"
	"strings"
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		text string
		cmd  string
		args int
		ok   bool
	}{
		{"/status", "status", 0, true},
		{"/Paths add /var/www", "paths", 2, true},
		{"/status@VpsAlertBot", "status", 0, true},
		{"/status@otherbot", "", 0, false},
		{"hello", "", 0, false},
	}
	for _, tt := range tests {
		cmd, args, ok := parseCommand(tt.text, "vpsalertbot")
		if cmd != tt.cmd || len(args) != tt.args || ok != tt.ok {
			t.Errorf("parseCommand(%q) = %q, %v, %v", tt.text, cmd, args, ok)
		}
	}
}

func TestAllowed(t *testing.T) {
	b := &Bot{Cfg: &config.Config{ChatID: -100, UserIDs: map[int64]bool{7: true}}}
	if !b.allowed(-100, 7) {
		t.Error("allowed user in allowed chat rejected")
	}
	if b.allowed(-100, 8) {
		t.Error("other group member accepted")
	}
	if b.allowed(7, 7) {
		t.Error("allowed user in another chat accepted")
	}

	open := &Bot{Cfg: &config.Config{ChatID: 42, UserIDs: map[int64]bool{}}}
	if !open.allowed(42, 42) || open.allowed(43, 43) {
		t.Error("chat-only check wrong")
	}
}

func TestEveryCommandHasHelp(t *testing.T) {
	for _, l := range i18n.Langs {
		for _, c := range commands {
			for _, key := range []string{"help." + c.name, "help." + c.name + ".short"} {
				if got := i18n.T(l, key); got == key {
					t.Errorf("%s: missing %s", l, key)
				}
			}
			short := i18n.T(l, "help."+c.name+".short")
			if len(short) > 256 || strings.ContainsAny(short, "<>") {
				t.Errorf("%s: %s.short must be plain text under 256 bytes for the command menu", l, c.name)
			}
			if h := commandHelp(l, c.name); len(h) > 3000 {
				t.Errorf("%s: help for %s is %d bytes", l, c.name, len(h))
			}
		}
		if h := helpList(l); len(h) > 3000 {
			t.Errorf("%s: /help list is %d bytes", l, len(h))
		}
	}
}

func TestUsageErrorIsTranslated(t *testing.T) {
	err := usage(i18n.Err("bot.err.days"))
	if !errors.As(err, new(usageError)) {
		t.Fatal("usage error not detected")
	}
	if got := i18n.Text(i18n.UZ, err); got != i18n.T(i18n.UZ, "bot.err.days") {
		t.Errorf("Text = %q", got)
	}
}
