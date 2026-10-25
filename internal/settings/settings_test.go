package settings

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNextOccurrence(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	if got := NextOccurrence(now, "11:30"); !got.Equal(time.Date(2026, 10, 25, 11, 30, 0, 0, time.UTC)) {
		t.Errorf("later today: got %v", got)
	}
	if got := NextOccurrence(now, "03:00"); !got.Equal(time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("tomorrow: got %v", got)
	}
}

func TestAdvanceScheduleSkipsMissedRuns(t *testing.T) {
	prev := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	got := AdvanceSchedule(prev, now, "03:00", 3)
	want := time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStoreUpdatePersistsAndRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Update(func(s *Settings) error { s.Backup.Paths = []string{"/etc"}; return nil }); err != nil {
		t.Fatal(err)
	}
	st.Update(func(s *Settings) error { s.Backup.Paths = append(s.Backup.Paths, "/x"); return errTest })

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if p := reopened.Get().Backup.Paths; len(p) != 1 || p[0] != "/etc" {
		t.Errorf("paths = %v, want [/etc]", p)
	}
}

type testErr struct{}

func (testErr) Error() string { return "test" }

var errTest = testErr{}
