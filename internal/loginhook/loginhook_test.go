package loginhook

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/knownips"
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
