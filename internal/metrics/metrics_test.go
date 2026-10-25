package metrics

import (
	"context"
	"strings"
	"testing"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
)

func TestCollect(t *testing.T) {
	s, err := Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.MemTotal == 0 || s.Cores == 0 || len(s.Disks) == 0 {
		t.Errorf("incomplete snapshot: %+v", s)
	}
	out := s.Format(i18n.EN, "test")
	if !strings.Contains(out, "CPU") {
		t.Errorf("unexpected format: %s", out)
	}
	t.Log("\n" + out)
}

func TestBytes(t *testing.T) {
	for n, want := range map[uint64]string{512: "512 B", 1536: "1.5 KiB", 5 << 30: "5.0 GiB"} {
		if got := Bytes(n); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestTopProcesses(t *testing.T) {
	byCPU, byMem, err := TopProcesses(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(byMem) == 0 || len(byMem) > 5 || len(byCPU) > 5 {
		t.Fatalf("byCPU %d, byMem %d", len(byCPU), len(byMem))
	}
	for i := 1; i < len(byMem); i++ {
		if byMem[i].RSS > byMem[i-1].RSS {
			t.Error("byMem not sorted")
		}
	}
	for i := 1; i < len(byCPU); i++ {
		if byCPU[i].CPUPercent > byCPU[i-1].CPUPercent {
			t.Error("byCPU not sorted")
		}
	}
	if byMem[0].PID == 0 || byMem[0].MemPercent <= 0 {
		t.Errorf("incomplete process: %+v", byMem[0])
	}
}
