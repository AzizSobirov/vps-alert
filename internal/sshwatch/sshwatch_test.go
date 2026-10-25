package sshwatch

import (
	"strings"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
)

var now = time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)

func TestParseLine(t *testing.T) {
	tests := []struct {
		line string
		want Event
		ok   bool
	}{
		// journald short-unix, Debian/Ubuntu
		{"1761400000.123456 web sshd[812]: Failed password for root from 203.0.113.7 port 52344 ssh2",
			Event{Kind: kindFailed, User: "root", IP: "203.0.113.7", Port: "52344"}, true},
		{"1761400000.2 web sshd[812]: Invalid user admin from 203.0.113.8 port 40000",
			Event{Kind: kindInvalid, User: "admin", IP: "203.0.113.8", Port: "40000"}, true},
		{"1761400000.3 web sshd[812]: Failed password for invalid user admin from 203.0.113.8 port 40000 ssh2",
			Event{Kind: kindFailedInvalid, User: "admin", IP: "203.0.113.8", Port: "40000"}, true},
		// OpenSSH >= 9.8 logs authentication from sshd-session
		{"1761400000.4 web sshd-session[901]: Failed publickey for deploy from 2001:db8::1 port 51000 ssh2: ED25519 SHA256:abc",
			Event{Kind: kindFailed, User: "deploy", IP: "2001:db8::1", Port: "51000"}, true},
		{"1761400000.5 web sshd[812]: Connection closed by authenticating user root 198.51.100.4 port 6000 [preauth]",
			Event{Kind: kindClosedPreauth, User: "root", IP: "198.51.100.4", Port: "6000"}, true},
		{"1761400000.6 web sshd[812]: Disconnected from invalid user test 198.51.100.5 port 6001 [preauth]",
			Event{Kind: kindClosedPreauth, User: "test", IP: "198.51.100.5", Port: "6001"}, true},
		// classic syslog, RHEL /var/log/secure, old OpenSSH without port
		{"Oct 25 14:03:11 web sshd[1201]: Invalid user oracle from 192.0.2.10",
			Event{Kind: kindInvalid, User: "oracle", IP: "192.0.2.10"}, true},
		// rsyslog ISO timestamps (Ubuntu 24.04)
		{"2026-10-25T14:03:11.123456+05:00 web sshd[1201]: Failed password for ubuntu from 192.0.2.11 port 22 ssh2",
			Event{Kind: kindFailed, User: "ubuntu", IP: "192.0.2.11", Port: "22"}, true},
		// not failures
		{"1761400000.7 web sshd[812]: Accepted publickey for root from 203.0.113.7 port 52344 ssh2: ED25519", Event{}, false},
		{"1761400000.8 web sshd[812]: Received disconnect from 203.0.113.9 port 1:11: Bye Bye [preauth]", Event{}, false},
		{"1761400000.9 web sshd[812]: pam_unix(sshd:auth): authentication failure; logname= uid=0 rhost=203.0.113.7 user=root", Event{}, false},
		{"1761400000.9 web cron[1]: Failed password for root from 1.2.3.4 port 1", Event{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseLine(tt.line, now)
		if ok != tt.ok {
			t.Errorf("ParseLine(%q) ok = %v", tt.line, ok)
			continue
		}
		got.Time = time.Time{}
		if got != tt.want {
			t.Errorf("ParseLine(%q) = %+v, want %+v", tt.line, got, tt.want)
		}
	}
}

func TestParseTime(t *testing.T) {
	if got := parseTime("1761400000.5 web sshd[1]: x", now); got.Unix() != 1761400000 || got.Nanosecond() != 500000000 {
		t.Errorf("unix: %v", got)
	}
	if got := parseTime("Oct 25 14:03:11 web sshd[1]: x", now); !got.Equal(time.Date(2026, 10, 25, 14, 3, 11, 0, time.UTC)) {
		t.Errorf("syslog: %v", got)
	}
	jan := time.Date(2027, 1, 1, 0, 30, 0, 0, time.UTC)
	if got := parseTime("Dec 31 23:59:00 web sshd[1]: x", jan); got.Year() != 2026 {
		t.Errorf("year rollover: %v", got)
	}
}

func TestAggregateCountsEachAttemptOnce(t *testing.T) {
	log := `1761400000 web sshd[1]: Invalid user admin from 203.0.113.8 port 40000
1761400001 web sshd[1]: Failed password for invalid user admin from 203.0.113.8 port 40000 ssh2
1761400002 web sshd[1]: Connection closed by invalid user admin 203.0.113.8 port 40000 [preauth]
1761400003 web sshd[1]: Failed password for root from 203.0.113.7 port 50000 ssh2
1761400004 web sshd[1]: Failed password for root from 203.0.113.7 port 50000 ssh2
1761400005 web sshd[1]: Connection closed by authenticating user root 203.0.113.7 port 50000 [preauth]
1761400006 web sshd[1]: Connection closed by authenticating user root 198.51.100.4 port 6000 [preauth]
1761399000 web sshd[1]: Failed password for root from 192.0.2.1 port 1 ssh2`
	st := Aggregate(parseAll(strings.NewReader(log), now), time.Unix(1761400000, 0))

	// admin once, root twice (two passwords), the key-only connection once;
	// the line before the window is ignored.
	if st.Total != 4 {
		t.Errorf("Total = %d, want 4", st.Total)
	}
	if st.IPs[0] != (Count{"203.0.113.7", 2}) || len(st.IPs) != 3 {
		t.Errorf("IPs = %v", st.IPs)
	}
	if st.Users[0] != (Count{"root", 3}) || st.Users[1] != (Count{"admin", 1}) {
		t.Errorf("Users = %v", st.Users)
	}
}

func TestTopLimitsAndOrders(t *testing.T) {
	got := top(map[string]int{"a": 1, "b": 5, "c": 5, "d": 2, "e": 3, "f": 4, "g": 1})
	if len(got) != topN || got[0] != (Count{"b", 5}) || got[1] != (Count{"c", 5}) || got[4] != (Count{"d", 2}) {
		t.Errorf("top = %v", got)
	}
}

func TestShouldAlert(t *testing.T) {
	st := Stats{Total: 150}
	if !shouldAlert(st, 100, time.Time{}, now) {
		t.Error("first alert not sent")
	}
	if shouldAlert(st, 100, now.Add(-30*time.Minute), now) {
		t.Error("alerted twice within an hour")
	}
	if !shouldAlert(st, 100, now.Add(-time.Hour), now) {
		t.Error("no alert after an hour")
	}
	if shouldAlert(st, 200, time.Time{}, now) || shouldAlert(st, 0, time.Time{}, now) {
		t.Error("alerted below threshold or while off")
	}
}

func TestFormat(t *testing.T) {
	for n, want := range map[int]string{7: "7", 1240: "1 240", 1234567: "1 234 567"} {
		if got := FormatCount(n); got != want {
			t.Errorf("FormatCount(%d) = %q", n, got)
		}
	}
	out := FormatTop(i18n.EN, Stats{Total: 3, IPs: []Count{{"1.2.3.4", 3}}, Users: []Count{{"<script>", 3}}})
	if !strings.Contains(out, "<code>1.2.3.4</code>: 3") || strings.Contains(out, "<script>") {
		t.Errorf("FormatTop = %q", out)
	}
}
