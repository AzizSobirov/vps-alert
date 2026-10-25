package knownips

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestAddAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if _, exists := Load(path); exists {
		t.Fatal("exists before the first write")
	}
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := Add(path, t0, "1.2.3.4", ""); err != nil {
		t.Fatal(err)
	}
	Add(path, t0.Add(-time.Hour), "1.2.3.4") // older sighting does not move it back
	ips, exists := Load(path)
	if !exists || len(ips) != 1 || !ips["1.2.3.4"].Equal(t0) {
		t.Errorf("got %v, %v", ips, exists)
	}
}

func TestKeepsMostRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < maxEntries+5; i++ {
		Add(path, t0.Add(time.Duration(i)*time.Minute), fmt.Sprintf("10.0.%d.%d", i/256, i%256))
	}
	ips, _ := Load(path)
	if len(ips) != maxEntries {
		t.Fatalf("len = %d, want %d", len(ips), maxEntries)
	}
	if _, ok := ips["10.0.0.0"]; ok {
		t.Error("the oldest address was kept")
	}
}
