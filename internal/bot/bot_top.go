package bot

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
)

func init() {
	registerCommand(command{name: "top", group: "monitor", example: "/top", doc: "alerts.md", run: (*Bot).top})
}

// top shows the processes using the most CPU and memory.
func (b *Bot) top(ctx context.Context, l i18n.Lang, args []string) error {
	text, err := topText(ctx, l)
	if err != nil {
		return err
	}
	b.reply(ctx, text, nil)
	return nil
}

func topText(ctx context.Context, l i18n.Lang) (string, error) {
	byCPU, byMem, err := metrics.TopProcesses(ctx, 10)
	if err != nil {
		return "", err
	}
	return formatTop(l, byCPU, byMem), nil
}

func formatTop(l i18n.Lang, byCPU, byMem []metrics.Process) string {
	var t strings.Builder
	t.WriteString(i18n.T(l, "top.title.cpu") + "\n")
	t.WriteString(processTable(l, byCPU))
	t.WriteString("\n" + i18n.T(l, "top.title.mem") + "\n")
	t.WriteString(processTable(l, byMem))
	return t.String()
}

// processTable renders processes as a monospace table. Names and users are
// cut short so rows stay on one line on a phone.
func processTable(l i18n.Lang, procs []metrics.Process) string {
	if len(procs) == 0 {
		return i18n.T(l, "top.none") + "\n"
	}
	var t strings.Builder
	t.WriteString("<pre>")
	fmt.Fprintf(&t, "%7s %6s %6s %-9s %s\n", "PID", "CPU%", "RAM%",
		html.EscapeString(cut(i18n.T(l, "top.col.user"), 9)), html.EscapeString(i18n.T(l, "top.col.name")))
	for _, p := range procs {
		fmt.Fprintf(&t, "%7d %6.1f %6.1f %-9s %s\n", p.PID, p.CPUPercent, p.MemPercent,
			html.EscapeString(cut(p.User, 9)), html.EscapeString(cut(p.Name, 20)))
	}
	t.WriteString("</pre>")
	return t.String()
}

func cut(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
