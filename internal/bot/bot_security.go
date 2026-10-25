package bot

import (
	"context"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/filewatch"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/sshwatch"
)

// Commands for the security watchers: /ssh, /files and /ports.

const defaultFailedLoginThreshold = 100

func init() {
	registerCommand(command{name: "ssh", group: "security", example: "/ssh threshold 50", doc: "security.md",
		run: (*Bot).sshCommand, state: (*Bot).sshState})
	registerCommand(command{name: "files", group: "security", example: "/files", doc: "security.md",
		run: (*Bot).filesCommand, state: (*Bot).filesState})
	registerCommand(command{name: "ports", group: "security", example: "/ports", doc: "security.md",
		run: (*Bot).portsCommand, state: (*Bot).portsState})
}

func (b *Bot) sshState(l i18n.Lang) string {
	if n := b.Store.Get().Security.FailedLoginThreshold; n > 0 {
		return i18n.T(l, "ssh.state_on", n)
	}
	return i18n.T(l, "ssh.state_off")
}

func (b *Bot) sshCommand(ctx context.Context, l i18n.Lang, args []string) error {
	switch {
	case len(args) == 0:
		b.reply(ctx, sshSummaryText(ctx, l)+"\n\n"+b.state(l, "ssh")+"\n\n"+commandHelp(l, "ssh"), nil)
		return nil
	case len(args) == 1 && (args[0] == "on" || args[0] == "off"):
		return b.saveSecurity(ctx, l, func(s *settings.Security) error {
			if args[0] == "off" {
				s.FailedLoginThreshold = 0
			} else if s.FailedLoginThreshold == 0 {
				s.FailedLoginThreshold = defaultFailedLoginThreshold
			}
			return nil
		})
	case len(args) == 2 && args[0] == "threshold":
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 || n > 100000 {
			return usage(i18n.Err("ssh.err.threshold"))
		}
		return b.saveSecurity(ctx, l, func(s *settings.Security) error { s.FailedLoginThreshold = n; return nil })
	}
	return usage(i18n.Err("bot.err.usage"))
}

// sshSummaryText describes failed SSH logins over the last 24 hours.
func sshSummaryText(ctx context.Context, l i18n.Lang) string {
	st, err := sshwatch.Summary(ctx, 24*time.Hour)
	if err != nil {
		return i18n.T(l, "ssh.unavailable", html.EscapeString(err.Error()))
	}
	text := i18n.T(l, "ssh.summary", sshwatch.FormatCount(st.Total))
	if top := sshwatch.FormatTop(l, st); top != "" {
		text += "\n" + top
	}
	return text
}

func (b *Bot) filesState(l i18n.Lang) string {
	if !b.Store.Get().Security.FileWatch {
		return i18n.T(l, "files.state_off")
	}
	watched, last, path := filewatch.Status(b.Cfg.DataDir)
	line := i18n.T(l, "files.state_on", watched) + "\n"
	if last.IsZero() {
		return line + i18n.T(l, "files.no_change")
	}
	return line + i18n.T(l, "files.last_change", last.Format(dateTime), html.EscapeString(path))
}

func (b *Bot) filesCommand(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "files")
		return nil
	}
	on, err := parseOnOff(args)
	if err != nil {
		return err
	}
	return b.saveSecurity(ctx, l, func(s *settings.Security) error { s.FileWatch = on; return nil })
}

func (b *Bot) portsState(l i18n.Lang) string {
	if b.Store.Get().Security.PortWatch {
		return i18n.T(l, "ports.state_on")
	}
	return i18n.T(l, "ports.state_off")
}

// maxPortLines keeps /ports well under Telegram's message size limit.
const maxPortLines = 60

func (b *Bot) portsCommand(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) > 0 {
		on, err := parseOnOff(args)
		if err != nil {
			return err
		}
		return b.saveSecurity(ctx, l, func(s *settings.Security) error { s.PortWatch = on; return nil })
	}
	b.reply(ctx, b.portsText(ctx, l)+"\n\n"+b.state(l, "ports")+"\n\n"+commandHelp(l, "ports"), nil)
	return nil
}

// portsText lists the listening ports.
func (b *Bot) portsText(ctx context.Context, l i18n.Lang) string {
	var t strings.Builder
	t.WriteString(i18n.T(l, "ports.title") + "\n")
	ports, err := filewatch.ListPorts(ctx)
	switch {
	case err != nil:
		t.WriteString(i18n.T(l, "ports.err", html.EscapeString(err.Error())))
	case len(ports) == 0:
		t.WriteString(i18n.T(l, "ports.none"))
	default:
		if len(ports) > maxPortLines {
			ports = ports[:maxPortLines]
		}
		t.WriteString(filewatch.FormatPorts(l, ports, filewatch.KnownPorts(b.Cfg.DataDir)))
	}
	return t.String()
}

func parseOnOff(args []string) (bool, error) {
	if len(args) == 1 && (args[0] == "on" || args[0] == "off") {
		return args[0] == "on", nil
	}
	return false, usage(i18n.Err("bot.err.usage"))
}

func (b *Bot) saveSecurity(ctx context.Context, l i18n.Lang, fn func(*settings.Security) error) error {
	if err := b.Store.Update(func(s *settings.Settings) error { return fn(&s.Security) }); err != nil {
		return err
	}
	b.reply(ctx, i18n.T(l, "bot.saved_msg"), nil)
	return nil
}
