package events

import (
	"testing"
	"time"
)

func TestAddAndCount(t *testing.T) {
	if Count(ServiceDown, time.Time{}) != 0 {
		t.Fatal("counted before SetDir")
	}
	Add(ServiceDown, 1) // no directory yet: ignored
	SetDir(t.TempDir())
	defer SetDir("")

	Add(ServiceDown, 2)
	Add(FileChange, 3)
	Add(NewPort, 0)
	if n := Count(ServiceDown, time.Now().Add(-time.Hour)); n != 2 {
		t.Errorf("service downs = %d, want 2", n)
	}
	if n := Count(FileChange, time.Now().Add(-time.Hour)); n != 3 {
		t.Errorf("file changes = %d, want 3", n)
	}
	if n := Count(ServiceDown, time.Now().Add(time.Minute)); n != 0 {
		t.Errorf("counted events before since: %d", n)
	}
}
