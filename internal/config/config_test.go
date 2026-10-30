package config

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vps-alert.env")
	os.WriteFile(path, []byte("# comment\nVPS_ALERT_T1=\"quoted value\"\nexport VPS_ALERT_T2=plain\nVPS_ALERT_T3=from-file\n"), 0o600)
	t.Setenv("VPS_ALERT_T3", "from-env")
	os.Unsetenv("VPS_ALERT_T1")
	os.Unsetenv("VPS_ALERT_T2")
	t.Cleanup(func() { os.Unsetenv("VPS_ALERT_T1"); os.Unsetenv("VPS_ALERT_T2") })

	if err := LoadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"VPS_ALERT_T1": "quoted value", "VPS_ALERT_T2": "plain", "VPS_ALERT_T3": "from-env"} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestParseIPList(t *testing.T) {
	nets, err := ParseIPList("1.2.3.4, 10.0.0.0/8, ::1")
	if err != nil {
		t.Fatal(err)
	}
	contains := func(ip string) bool {
		for _, n := range nets {
			if n.Contains(net.ParseIP(ip)) {
				return true
			}
		}
		return false
	}
	for ip, want := range map[string]bool{"1.2.3.4": true, "1.2.3.5": false, "10.9.8.7": true, "::1": true} {
		if contains(ip) != want {
			t.Errorf("%s: got %v, want %v", ip, !want, want)
		}
	}
	if _, err := ParseIPList("not-an-ip"); err == nil {
		t.Error("expected error")
	}
}

func TestAlertChatID(t *testing.T) {
	t.Setenv("BOT_TOKEN", "token")
	t.Setenv("ALLOWED_CHAT_ID", "123")
	t.Setenv("ALERT_CHAT_ID", "")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AlertChatID != 123 {
		t.Errorf("AlertChatID = %d, want ALLOWED_CHAT_ID", c.AlertChatID)
	}

	t.Setenv("ALERT_CHAT_ID", "-1001234567890")
	if c, err = Load(); err != nil || c.AlertChatID != -1001234567890 || c.ChatID != 123 {
		t.Errorf("ALERT_CHAT_ID not used: %+v, %v", c, err)
	}

	t.Setenv("ALERT_CHAT_ID", "@channel")
	if _, err := Load(); err == nil {
		t.Error("invalid ALERT_CHAT_ID accepted")
	}
}
