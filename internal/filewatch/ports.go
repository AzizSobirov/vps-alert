package filewatch

import (
	"context"
	"fmt"
	"html"
	"net"
	"sort"
	"strconv"
	"strings"
	"syscall"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
)

// Port is a listening socket.
type Port struct {
	Proto   string `json:"proto"` // tcp or udp
	Addr    string `json:"addr"`  // host:port
	Process string `json:"process,omitempty"`
}

func (p Port) key() string { return p.Proto + " " + p.Addr }

// UDP sockets in the ephemeral range are usually short-lived clients (DNS,
// NTP), not services, and would make the port watch noisy.
func ephemeralUDP(port uint32) bool { return port >= 32768 && port <= 60999 }

// ListPorts returns the listening TCP sockets and bound UDP sockets.
func ListPorts(ctx context.Context) ([]Port, error) {
	conns, err := gnet.ConnectionsWithContext(ctx, "inet")
	if err != nil {
		return nil, err
	}
	names := map[int32]string{}
	seen := map[string]bool{}
	var out []Port
	for _, c := range conns {
		var proto string
		switch {
		case c.Type == syscall.SOCK_STREAM && c.Status == "LISTEN":
			proto = "tcp"
		case c.Type == syscall.SOCK_DGRAM && c.Raddr.IP == "" && c.Raddr.Port == 0 && !ephemeralUDP(c.Laddr.Port):
			proto = "udp"
		default:
			continue
		}
		p := Port{Proto: proto, Addr: net.JoinHostPort(c.Laddr.IP, strconv.Itoa(int(c.Laddr.Port)))}
		if seen[p.key()] {
			continue
		}
		seen[p.key()] = true
		if c.Pid > 0 {
			name, ok := names[c.Pid]
			if !ok {
				if proc, err := process.NewProcessWithContext(ctx, c.Pid); err == nil {
					name, _ = proc.NameWithContext(ctx)
				}
				names[c.Pid] = name
			}
			p.Process = name
		}
		out = append(out, p)
	}
	sortPorts(out)
	return out, nil
}

func sortPorts(ps []Port) {
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Proto != ps[j].Proto {
			return ps[i].Proto < ps[j].Proto
		}
		return portNum(ps[i].Addr) < portNum(ps[j].Addr) ||
			(portNum(ps[i].Addr) == portNum(ps[j].Addr) && ps[i].Addr < ps[j].Addr)
	})
}

func portNum(addr string) int {
	_, p, _ := net.SplitHostPort(addr)
	n, _ := strconv.Atoi(p)
	return n
}

// portTracker decides which ports are new. A port is reported only when it
// is still listening on the check after it first appeared, so short-lived
// sockets do not alert. Ports once accepted stay known even while closed,
// so a service restart never looks like a new port.
type portTracker struct {
	known   map[string]Port
	pending map[string]bool
}

func (t *portTracker) reset(cur []Port) {
	if t.known == nil {
		t.known = map[string]Port{}
	}
	for _, p := range cur {
		t.known[p.key()] = p
	}
	t.pending = nil
}

// update returns the ports that became listening and stayed open.
func (t *portTracker) update(cur []Port) []Port {
	if t.pending == nil {
		t.pending = map[string]bool{}
	}
	stillNew := map[string]bool{}
	var opened []Port
	for _, p := range cur {
		k := p.key()
		if _, ok := t.known[k]; ok {
			continue
		}
		if t.pending[k] {
			t.known[k] = p
			opened = append(opened, p)
			continue
		}
		stillNew[k] = true
	}
	t.pending = stillNew
	return opened
}

func samePorts(a, b map[string]Port) bool {
	if len(a) != len(b) || (a == nil) != (b == nil) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func clonePorts(m map[string]Port) map[string]Port {
	if m == nil {
		return nil
	}
	out := make(map[string]Port, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// FormatPorts renders ports, one per line. Ports not in known are marked as
// new; known ports that are no longer listening are listed as closed. Pass a
// nil known map to skip both markers.
func FormatPorts(l i18n.Lang, ports []Port, known map[string]Port) string {
	var b strings.Builder
	open := map[string]bool{}
	for i, p := range ports {
		open[p.key()] = true
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(formatPort(p))
		if known != nil {
			if _, ok := known[p.key()]; !ok {
				b.WriteString(" " + i18n.T(l, "ports.new"))
			}
		}
	}
	if known != nil {
		var closed []Port
		for k, p := range known {
			if !open[k] {
				closed = append(closed, p)
			}
		}
		sortPorts(closed)
		for _, p := range closed {
			b.WriteString("\n" + formatPort(p) + " " + i18n.T(l, "ports.closed"))
		}
	}
	return b.String()
}

func formatPort(p Port) string {
	proc := p.Process
	if proc == "" {
		proc = "?"
	}
	return fmt.Sprintf("%s <code>%s</code> (%s)", p.Proto, html.EscapeString(p.Addr), html.EscapeString(proc))
}
