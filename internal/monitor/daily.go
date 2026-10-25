package monitor

import (
	"context"
	"fmt"
	"html"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/events"
	"github.com/AzizSobirov/vps-alert/internal/httpcheck"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/services"
	"github.com/AzizSobirov/vps-alert/internal/sshwatch"
	"github.com/AzizSobirov/vps-alert/internal/trust"
)

// The daily report summarizes the last day: SSH logins, services, websites
// and security events. A quiet day gives a short report; problems stand out
// with ⚠️ and 🔴.

const reportPeriod = 24 * time.Hour

// Names listed per line before the rest is cut off.
const maxNames = 5

// dailySummary returns the report's sections after the server status and
// backups, separated by blank lines.
func (m *Monitor) dailySummary(ctx context.Context, l i18n.Lang, now time.Time) string {
	s := m.Store.Get()
	since := now.Add(-reportPeriod)
	var sections []string

	if st, err := sshwatch.Summary(ctx, reportPeriod); err == nil {
		var sshd *sshwatch.SSHDConfig
		if c, err := sshwatch.ReadSSHDConfig(ctx); err == nil {
			sshd = &c
		}
		trusted := func(lg sshwatch.Login) bool {
			return trust.Match(s.Security.LoginWhitelist, lg.User, lg.IP) || inNets(lg.IP, m.EnvWhitelist)
		}
		sections = append(sections, sshSection(l, st, sshd, trusted))
	}

	var lines []string
	if items := services.Selected(s.Services); len(items) > 0 {
		var down []string
		for _, it := range items {
			if st := services.Check(ctx, it); !st.OK {
				down = append(down, it.Label())
			}
		}
		lines = append(lines, servicesLine(l, len(items), down, events.Count(events.ServiceDown, since)))
	}
	if len(s.HTTPChecks) > 0 {
		var down []string
		for _, c := range s.HTTPChecks {
			if st, ok := httpcheck.Snapshot(c.URL); ok && !st.Up {
				down = append(down, c.URL)
			}
		}
		lines = append(lines, sitesLine(l, len(s.HTTPChecks), down, events.Count(events.SiteDown, since)))
	}
	if s.Security.FileWatch || s.Security.PortWatch {
		lines = append(lines, securityLine(l, events.Count(events.FileChange, since), events.Count(events.NewPort, since)))
	}
	if len(lines) > 0 {
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

// sshSection summarizes the logins of the day. Trusted logins, such as a CI
// deploy user, are counted on their own line so they do not drown out the
// ones that matter.
func sshSection(l i18n.Lang, st sshwatch.Stats, sshd *sshwatch.SSHDConfig, trusted func(sshwatch.Login) bool) string {
	people, robots := map[string]int{}, map[string]int{}
	passwords := 0
	for _, lg := range st.Logins {
		if lg.PasswordLike() {
			passwords++
		}
		if trusted(lg) {
			robots[lg.User]++
		} else {
			people[lg.User]++
		}
	}

	var b strings.Builder
	b.WriteString(i18n.T(l, "monitor.report.ssh_title"))
	if n := sum(people); n > 0 {
		b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_logins", n, countList(people)))
	} else {
		b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_no_logins"))
	}
	for _, c := range sortedCounts(robots) {
		b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_trusted", html.EscapeString(c.name), c.n))
	}
	b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_failed", sshwatch.FormatCount(st.Total)))
	if passwords > 0 {
		b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_password_logins", passwords))
	}
	if sshd != nil {
		if sshd.PasswordAuth {
			b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_password_on"))
		}
		if sshd.PermitRoot == "yes" {
			b.WriteString("\n" + i18n.T(l, "monitor.report.ssh_root_on"))
		}
	}
	return b.String()
}

func servicesLine(l i18n.Lang, total int, down []string, stops int) string {
	line := i18n.T(l, "monitor.report.services_ok", total)
	if len(down) > 0 {
		line = i18n.T(l, "monitor.report.services_down", html.EscapeString(nameList(down)))
	}
	if stops > 0 {
		line += i18n.T(l, "monitor.report.services_stops", stops)
	}
	return line
}

func sitesLine(l i18n.Lang, total int, down []string, outages int) string {
	line := i18n.T(l, "monitor.report.sites", total-len(down), total)
	if len(down) > 0 {
		line += " · 🔴 " + html.EscapeString(nameList(down))
	}
	if outages > 0 {
		line += i18n.T(l, "monitor.report.sites_outages", outages)
	}
	return line
}

func securityLine(l i18n.Lang, files, ports int) string {
	if files == 0 && ports == 0 {
		return i18n.T(l, "monitor.report.security_none")
	}
	return i18n.T(l, "monitor.report.security", files, ports)
}

type nameCount struct {
	name string
	n    int
}

func sortedCounts(m map[string]int) []nameCount {
	out := make([]nameCount, 0, len(m))
	for k, n := range m {
		out = append(out, nameCount{k, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].name < out[j].name
	})
	return out
}

// countList renders "aziz ×2, root ×1".
func countList(m map[string]int) string {
	var parts []string
	for i, c := range sortedCounts(m) {
		if i == maxNames {
			parts = append(parts, "…")
			break
		}
		parts = append(parts, fmt.Sprintf("%s ×%d", html.EscapeString(c.name), c.n))
	}
	return strings.Join(parts, ", ")
}

func nameList(names []string) string {
	if len(names) > maxNames {
		names = append(names[:maxNames:maxNames], "…")
	}
	return strings.Join(names, ", ")
}

func sum(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func inNets(host string, nets []*net.IPNet) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
