package filewatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
)

const (
	edKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICT9eCxgUwIWmgwr8uBMXQtdJYYTiwrldHKwPmtgNn9y test@example"
	edFP   = "SHA256:j4bhX0W+JGIpAlh1rFmC80zrN/pcjTWFFbLrkfl9kqU" // from ssh-keygen -lf
	rsaKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQC1x9yRwwdvFeUGfRoKFhPTpbxR7E6PamlTNpXCBFPrwQwZtJB+wokRGauBgmkMUG2/li2tgu5r3BPmpHc6mvl3LH5f2x5E3NRhxJqld2AurMGr/zfw6saqnliwEZiVZLnfWTIUSLjSqEjttLAKabbhYb/XyCdmXrh46x5/heC3BQ== rsa key"
	rsaFP  = "SHA256:qtIAXBcdfBkbyUcyWAPm//aNBFslGh/7xZ9B+7egAG8"
)

func TestParseKeyFingerprint(t *testing.T) {
	typ, fp, comment, ok := parseKey(edKey)
	if !ok || typ != "ssh-ed25519" || fp != edFP || comment != "test@example" {
		t.Errorf("ed25519: %q %q %q %v", typ, fp, comment, ok)
	}
	// Options before the key and a comment with spaces.
	_, fp, comment, ok = parseKey(`command="/bin/true",no-pty ` + rsaKey)
	if !ok || fp != rsaFP || comment != "rsa key" {
		t.Errorf("rsa with options: %q %q %v", fp, comment, ok)
	}
	if _, _, _, ok := parseKey("# just a comment"); ok {
		t.Error("comment parsed as key")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func kinds(changes []Change) string {
	var s []string
	for _, c := range changes {
		s = append(s, filepath.Base(c.Path)+":"+c.Kind)
	}
	return strings.Join(s, " ")
}

func TestDiffCreateModifyDeleteChmod(t *testing.T) {
	dir := t.TempDir()
	patterns := []string{filepath.Join(dir, "cron.d", "*"), filepath.Join(dir, "sshd_config")}
	write(t, filepath.Join(dir, "sshd_config"), "PermitRootLogin no\n")
	write(t, filepath.Join(dir, "cron.d", "backup"), "0 3 * * * root /backup\n")
	base := Scan(patterns)
	if len(base) != 2 {
		t.Fatalf("scanned %d files", len(base))
	}
	if c := Diff(base, Scan(patterns)); len(c) != 0 {
		t.Fatalf("no change reported as %s", kinds(c))
	}

	write(t, filepath.Join(dir, "sshd_config"), "PermitRootLogin yes\n")
	write(t, filepath.Join(dir, "cron.d", "miner"), "* * * * * root /tmp/x\n")
	os.Remove(filepath.Join(dir, "cron.d", "backup"))
	got := kinds(Diff(base, Scan(patterns)))
	if got != "backup:deleted miner:created sshd_config:modified" {
		t.Errorf("changes = %s", got)
	}

	base = Scan(patterns)
	os.Chmod(filepath.Join(dir, "sshd_config"), 0o666)
	c := Diff(base, Scan(patterns))
	if len(c) != 1 || c[0].Kind != "perms" || !strings.HasPrefix(c[0].Old, "0644 ") || !strings.HasPrefix(c[0].New, "0666 ") {
		t.Errorf("chmod: %+v", c)
	}
}

func TestPasswdGroupShadowDetails(t *testing.T) {
	dir := t.TempDir()
	passwd, group, shadow := filepath.Join(dir, "passwd"), filepath.Join(dir, "group"), filepath.Join(dir, "shadow")
	patterns := []string{passwd, group, shadow}
	write(t, passwd, "root:x:0:0:root:/root:/bin/bash\nbob:x:1000:1000::/home/bob:/bin/bash\nold:x:1001:1001::/home/old:/bin/sh\n")
	write(t, group, "root:x:0:\nsudo:x:27:bob\nusers:x:100:bob\n")
	write(t, shadow, "root:$6$aaa:19000:0:99999:7:::\nbob:$6$bbb:19000:0:99999:7:::\n")
	base := Scan(patterns)

	write(t, passwd, "root:x:0:0:root:/root:/bin/bash\nbob:x:1000:1000::/home/bob:/bin/bash\nmallory:x:1002:1002::/home/m:/bin/sh\ntoor:x:0:0::/root:/bin/sh\n")
	write(t, group, "root:x:0:\nsudo:x:27:bob,mallory\nusers:x:100:bob,mallory\n")
	write(t, shadow, "root:$6$aaa:19000:0:99999:7:::\nbob:$6$NEW:19300:0:99999:7:::\n")

	out := FormatChanges(i18n.EN, Diff(base, Scan(patterns)))
	for _, want := range []string{
		"New user: <code>mallory</code> (uid 1002)",
		"root rights (uid 0): <code>toor</code>",
		"User removed: <code>old</code>",
		"<code>mallory</code> added to group <b>sudo</b>",
		"changed for: bob",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// "users" is not a privileged group, and no password hash may leak.
	if strings.Contains(out, "group <b>users</b>") || strings.Contains(out, "$6$") {
		t.Errorf("unexpected detail:\n%s", out)
	}
	if strings.Contains(string(mustJSON(t, Scan(patterns))), "$6$") {
		t.Error("shadow hashes stored in the baseline")
	}
}

func TestAuthorizedKeysDetails(t *testing.T) {
	dir := t.TempDir()
	keys := filepath.Join(dir, "root", ".ssh", "authorized_keys")
	patterns := []string{filepath.Join(dir, "*", ".ssh", "authorized_keys")}
	write(t, keys, rsaKey+"\n")
	base := Scan(patterns)
	write(t, keys, edKey+"\n")

	out := FormatChanges(i18n.EN, Diff(base, Scan(patterns)))
	if !strings.Contains(out, "New SSH key: <code>"+edFP+"</code> ssh-ed25519 test@example") ||
		!strings.Contains(out, "SSH key removed: <code>"+rsaFP+"</code> ssh-rsa rsa key") {
		t.Errorf("keys diff:\n%s", out)
	}
}

func TestPortTracker(t *testing.T) {
	ssh := Port{Proto: "tcp", Addr: "0.0.0.0:22", Process: "sshd"}
	redis := Port{Proto: "tcp", Addr: "0.0.0.0:6379", Process: "redis-server"}
	brief := Port{Proto: "udp", Addr: "0.0.0.0:5353"}

	tr := &portTracker{}
	tr.reset([]Port{ssh})

	if got := tr.update([]Port{ssh, redis, brief}); len(got) != 0 {
		t.Errorf("alerted on first sight: %v", got)
	}
	// brief closed again: never reported; redis stayed: reported once.
	if got := tr.update([]Port{ssh, redis}); len(got) != 1 || got[0] != redis {
		t.Errorf("second check: %v", got)
	}
	if got := tr.update([]Port{ssh, redis}); len(got) != 0 {
		t.Errorf("reported twice: %v", got)
	}
	// A restart closes and reopens the port: still known, no alert.
	tr.update([]Port{ssh})
	if got := tr.update([]Port{ssh, redis}); len(got) != 0 {
		t.Errorf("restart reported: %v", got)
	}

	out := FormatPorts(i18n.EN, []Port{ssh, brief}, tr.known)
	if !strings.Contains(out, "0.0.0.0:5353</code> (?) 🆕") || !strings.Contains(out, "0.0.0.0:6379</code> (redis-server) (closed)") {
		t.Errorf("FormatPorts:\n%s", out)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestBaselineRoundTrip(t *testing.T) {
	dir := t.TempDir()
	b := baseline{Files: map[string]FileState{"/etc/passwd": {SHA256: "x", Mode: 0o644}}, Ports: map[string]Port{"tcp 0.0.0.0:22": {Proto: "tcp", Addr: "0.0.0.0:22"}}}
	if err := b.save(dir); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(baselinePath(dir)); fi.Mode().Perm() != 0o600 {
		t.Errorf("baseline mode %v", fi.Mode().Perm())
	}
	got := loadBaseline(dir)
	if got.Files["/etc/passwd"].SHA256 != "x" || !samePorts(got.Ports, b.Ports) {
		t.Errorf("round trip: %+v", got)
	}
	if loadBaseline(t.TempDir()).Files != nil {
		t.Error("missing baseline should be nil")
	}
}
