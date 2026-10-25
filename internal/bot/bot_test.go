package bot

import (
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/config"
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
