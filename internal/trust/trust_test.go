package trust

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"92.63.204.12":     "92.63.204.12",
		" 10.0.0.7/24 ":    "10.0.0.0/24",
		"2001:DB8::1":      "2001:db8::1",
		"deploy":           "deploy",
		"github_runner-01": "github_runner-01",
	} {
		if got, err := Normalize(in); err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "1.2.3.4/99", "Root", "a b", "rm -rf", "-x"} {
		if _, err := Normalize(bad); err == nil {
			t.Errorf("Normalize(%q) accepted", bad)
		}
	}
}

func TestMatch(t *testing.T) {
	entries := []string{"92.63.204.12", "10.0.0.0/24", "deploy"}
	for _, tt := range []struct {
		user, host string
		want       bool
	}{
		{"aziz", "92.63.204.12", true},
		{"aziz", "10.0.0.99", true},
		{"aziz", "10.0.1.1", false},
		{"deploy", "20.42.0.1", true}, // any address for a trusted user
		{"root", "20.42.0.1", false},
		{"deploy", "", true},
		{"aziz", "", false},
	} {
		if got := Match(entries, tt.user, tt.host); got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.user, tt.host, got, tt.want)
		}
	}
}
