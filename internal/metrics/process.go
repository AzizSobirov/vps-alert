package metrics

import (
	"context"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type Process struct {
	PID        int32
	Name       string
	User       string
	CPUPercent float64 // of one core, like top: a busy 4-thread process can show 400
	MemPercent float64
	RSS        uint64
}

// cpuSample is how long TopProcesses measures CPU usage. A lifetime average
// would hide a process that only just started spinning.
const cpuSample = time.Second

// TopProcesses returns the n processes using the most CPU and the n using
// the most memory. It takes about one second. Processes that cannot be
// read (they exited, or belong to another user when not root) are skipped.
func TopProcesses(ctx context.Context, n int) (byCPU, byMem []Process, err error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	before := map[int32]float64{}
	for _, p := range procs {
		if t, err := p.TimesWithContext(ctx); err == nil {
			before[p.Pid] = t.User + t.System
		}
	}
	start := time.Now()
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(cpuSample):
	}
	elapsed := time.Since(start).Seconds()

	type sample struct {
		p   *process.Process
		cpu float64
		rss uint64
	}
	var all []sample
	for _, p := range procs {
		s := sample{p: p}
		if t0, ok := before[p.Pid]; ok {
			if t, err := p.TimesWithContext(ctx); err == nil && t.User+t.System >= t0 {
				s.cpu = (t.User + t.System - t0) / elapsed * 100
			}
		}
		if mi, err := p.MemoryInfoWithContext(ctx); err == nil {
			s.rss = mi.RSS
		}
		if s.cpu > 0 || s.rss > 0 {
			all = append(all, s)
		}
	}

	// Names and users cost a lookup each, so only the winners get them.
	describe := func(s sample) Process {
		pr := Process{PID: s.p.Pid, CPUPercent: s.cpu, RSS: s.rss}
		if vm.Total > 0 {
			pr.MemPercent = float64(s.rss) / float64(vm.Total) * 100
		}
		pr.Name, _ = s.p.NameWithContext(ctx)
		pr.User, _ = s.p.UsernameWithContext(ctx)
		return pr
	}

	sort.Slice(all, func(i, j int) bool { return all[i].cpu > all[j].cpu })
	for _, s := range all {
		if len(byCPU) == n || s.cpu <= 0 {
			break
		}
		byCPU = append(byCPU, describe(s))
	}
	sort.Slice(all, func(i, j int) bool { return all[i].rss > all[j].rss })
	for _, s := range all {
		if len(byMem) == n || s.rss == 0 {
			break
		}
		byMem = append(byMem, describe(s))
	}
	return byCPU, byMem, nil
}
