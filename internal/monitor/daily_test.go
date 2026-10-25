package monitor

import (
	"strings"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/sshwatch"
)

func TestSSHSection(t *testing.T) {
	now := time.Now()
	st := sshwatch.Stats{Total: 1163, Logins: []sshwatch.Login{
		{Time: now, User: "aziz", IP: "92.63.204.12", Method: "publickey"},
		{Time: now, User: "aziz", IP: "92.63.204.12", Method: "publickey"},
		{Time: now, User: "root", IP: "10.0.0.5", Method: "password"},
	}}
	for i := 0; i < 14; i++ {
		st.Logins = append(st.Logins, sshwatch.Login{Time: now, User: "deploy", IP: "20.42.0.1", Method: "publickey"})
	}
	trusted := func(lg sshwatch.Login) bool { return lg.User == "deploy" }
	out := sshSection(i18n.EN, st, &sshwatch.SSHDConfig{PasswordAuth: true, PermitRoot: "prohibit-password"}, trusted)

	for _, want := range []string{
		"Logins: 3: aziz ×2, root ×1",
		"deploy: 14 times (trusted",
		"Failed attempts: 1 163",
		"Logins with a password: 1",
		"Password login is allowed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Root can log in") {
		t.Error("root warning shown for prohibit-password")
	}
}

func TestSSHSectionQuietDay(t *testing.T) {
	out := sshSection(i18n.UZ, sshwatch.Stats{}, &sshwatch.SSHDConfig{PermitRoot: "no"}, func(sshwatch.Login) bool { return false })
	if !strings.Contains(out, "Kirishlar yo'q") || strings.Contains(out, "⚠️") {
		t.Errorf("quiet day:\n%s", out)
	}
}

func TestStatusLines(t *testing.T) {
	if got := servicesLine(i18n.EN, 3, nil, 0); got != "⚙️ Services: all 3 running" {
		t.Errorf("services ok: %q", got)
	}
	if got := servicesLine(i18n.EN, 3, []string{"nginx"}, 2); !strings.Contains(got, "🔴 nginx") || !strings.Contains(got, "stops in 24h: 2") {
		t.Errorf("services down: %q", got)
	}
	if got := sitesLine(i18n.EN, 2, []string{"https://a.uz"}, 1); !strings.Contains(got, "1/2 up") || !strings.Contains(got, "🔴 https://a.uz") {
		t.Errorf("sites: %q", got)
	}
	if got := securityLine(i18n.EN, 0, 0); !strings.Contains(got, "no file changes") {
		t.Errorf("security quiet: %q", got)
	}
	if got := securityLine(i18n.EN, 2, 1); !strings.Contains(got, "file changes: 2, new ports: 1") {
		t.Errorf("security: %q", got)
	}
	if got := nameList([]string{"a", "b", "c", "d", "e", "f", "g"}); got != "a, b, c, d, e, …" {
		t.Errorf("nameList: %q", got)
	}
}
