// Package metrics collects a point-in-time snapshot of host resources.
package metrics

import (
	"context"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
)

type Disk struct {
	Mount       string
	Total, Used uint64
	// Free is the space still available to ordinary users; Used+Free is
	// the point at which the disk counts as full (root's reserved blocks
	// excluded).
	Free         uint64
	Percent      float64
	InodePercent float64
}

type Snapshot struct {
	Hostname   string
	OS         string
	Uptime     time.Duration
	Cores      int
	CPUPercent float64
	Load1      float64
	Load5      float64
	Load15     float64
	MemTotal   uint64
	MemUsed    uint64
	MemPercent float64
	SwapTotal  uint64
	SwapUsed   uint64
	Disks      []Disk
}

// Pseudo and read-only image filesystems that are always "full" or irrelevant.
var skipFS = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "overlay": true, "squashfs": true,
	"proc": true, "sysfs": true, "cgroup": true, "cgroup2": true, "devfs": true,
	"autofs": true, "nsfs": true, "fuse.lxcfs": true, "efivarfs": true,
}

// Collect samples CPU usage over one second, so it takes about that long.
func Collect(ctx context.Context) (Snapshot, error) {
	var s Snapshot
	s.Hostname, _ = os.Hostname()

	if info, err := host.InfoWithContext(ctx); err == nil {
		s.OS = strings.TrimSpace(info.Platform + " " + info.PlatformVersion)
		s.Uptime = time.Duration(info.Uptime) * time.Second
	}

	s.Cores, _ = cpu.CountsWithContext(ctx, true)
	pct, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return s, fmt.Errorf("cpu: %w", err)
	}
	if len(pct) > 0 {
		s.CPUPercent = pct[0]
	}

	if l, err := load.AvgWithContext(ctx); err == nil {
		s.Load1, s.Load5, s.Load15 = l.Load1, l.Load5, l.Load15
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return s, fmt.Errorf("memory: %w", err)
	}
	s.MemTotal, s.MemUsed, s.MemPercent = vm.Total, vm.Used, vm.UsedPercent
	if sw, err := mem.SwapMemoryWithContext(ctx); err == nil {
		s.SwapTotal, s.SwapUsed = sw.Total, sw.Used
	}

	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return s, fmt.Errorf("disk: %w", err)
	}
	// Bind mounts, btrfs subvolumes and APFS volumes show the same storage
	// under several mountpoints; identical total and used bytes give them away.
	seen := map[string]bool{}
	for _, p := range parts {
		if skipFS[p.Fstype] || seen[p.Device] || strings.HasPrefix(p.Mountpoint, "/snap/") {
			continue
		}
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		size := fmt.Sprintf("%d/%d", u.Total, u.Used)
		if seen[size] {
			continue
		}
		seen[p.Device], seen[size] = true, true
		s.Disks = append(s.Disks, Disk{
			Mount:        p.Mountpoint,
			Total:        u.Total,
			Used:         u.Used,
			Free:         u.Free,
			Percent:      u.UsedPercent,
			InodePercent: u.InodesUsedPercent,
		})
	}
	return s, nil
}

// Format renders the snapshot as a Telegram HTML message in l.
func (s Snapshot) Format(l i18n.Lang, server string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🖥 <b>%s</b>\n", html.EscapeString(server))
	if s.OS != "" {
		b.WriteString(i18n.T(l, "metrics.os_uptime", html.EscapeString(s.OS), FormatDuration(s.Uptime)) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(i18n.T(l, "metrics.cpu",
		dot(s.CPUPercent, 70, 90), s.CPUPercent, s.Cores, s.Load1, s.Load5, s.Load15) + "\n")
	b.WriteString(i18n.T(l, "metrics.ram",
		dot(s.MemPercent, 75, 90), s.MemPercent, Bytes(s.MemUsed), Bytes(s.MemTotal)) + "\n")
	if s.SwapTotal > 0 {
		b.WriteString(i18n.T(l, "metrics.swap", Bytes(s.SwapUsed), Bytes(s.SwapTotal)) + "\n")
	}
	for _, d := range s.Disks {
		b.WriteString(i18n.T(l, "metrics.disk",
			dot(d.Percent, 75, 85), html.EscapeString(d.Mount), d.Percent, Bytes(d.Used), Bytes(d.Total), d.InodePercent) + "\n")
	}
	return b.String()
}

func dot(v, warn, crit float64) string {
	switch {
	case v >= crit:
		return "🔴"
	case v >= warn:
		return "🟡"
	default:
		return "🟢"
	}
}

func Bytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}
