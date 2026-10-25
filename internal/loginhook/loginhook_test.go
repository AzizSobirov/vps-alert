package loginhook

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/knownips"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

func TestNewAddress(t *testing.T) {
	path := filepath.Join(t.TempDir(), knownips.FileName)
	now := time.Now()

	// Before the service has seeded the list nothing is flagged or written.
	if NewAddress(path, "1.2.3.4", now) {
		t.Error("flagged before the list exists")
	}
	if _, exists := knownips.Load(path); exists {
		t.Fatal("the PAM hook created the list")
	}

	knownips.Add(path, now, "1.2.3.4")
	if NewAddress(path, "1.2.3.4", now) {
		t.Error("a known address was flagged")
	}
	if !NewAddress(path, "5.6.7.8", now) {
		t.Error("a new address was not flagged")
	}
	if NewAddress(path, "5.6.7.8", now) {
		t.Error("the same address was flagged twice")
	}
}

func TestMessageTrust(t *testing.T) {
	dir := t.TempDir()
	envNets, _ := config.ParseIPList("203.0.113.0/24")
	cfg := &config.Config{ServerName: "web", DataDir: dir, LoginWhitelist: envNets}
	knownips.Add(filepath.Join(dir, knownips.FileName), time.Now(), "92.63.204.12")
	var s settings.Settings
	s.Security.LoginWhitelist = []string{"deploy"}
	now := time.Now()

	// The CI deploy user from a random GitHub address: silent, no button,
	// not marked new, and its address is not remembered.
	text, kb, silent := Message(cfg, s, Event{User: "deploy", Host: "20.42.11.7", Service: "sshd"}, now)
	if !silent || kb != nil || strings.Contains(text, "🆕") {
		t.Errorf("trusted user: silent=%v kb=%v text=%q", silent, kb, text)
	}
	if known, _ := knownips.Load(filepath.Join(dir, knownips.FileName)); len(known) != 1 {
		t.Errorf("a trusted user's address was remembered: %v", known)
	}

	// An address whitelisted in the env file is reported silently, not hidden.
	if text, _, silent := Message(cfg, s, Event{User: "aziz", Host: "203.0.113.9"}, now); !silent || text == "" {
		t.Error("env whitelist: login not sent silently")
	}

	// A stranger: loud, marked new, with a button to trust the address.
	text, kb, silent = Message(cfg, s, Event{User: "root", Host: "45.148.10.240"}, now)
	if silent || !strings.Contains(text, "🆕") || kb == nil || kb.InlineKeyboard[0][0].CallbackData != "trust:45.148.10.240" {
		t.Errorf("stranger: silent=%v kb=%+v text=%q", silent, kb, text)
	}

	// A local console login has no address to trust.
	if _, kb, _ := Message(cfg, s, Event{User: "root"}, now); kb != nil {
		t.Error("local login got a trust button")
	}
}
