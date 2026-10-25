package bot

import (
	"context"
	"fmt"
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
	sec := b.Store.Get().Security
	line := i18n.T(l, "ssh.state_off")
	if sec.FailedLoginThreshold > 0 {
		line = i18n.T(l, "ssh.state_on", sec.FailedLoginThreshold)
	}
	if sec.PasswordLoginAlert {
		return line + "\n" + i18n.T(l, "ssh.state_pw_on")
	}
	return line + "\n" + i18n.T(l, "ssh.state_pw_off")
}

func (b *Bot) sshCommand(ctx context.Context, l i18n.Lang, args []string) error {
	switch {
	case len(args) == 0:
		b.reply(ctx, sshSummaryText(ctx, l)+"\n\n"+b.state(l, "ssh")+"\n\n"+commandHelp(l, "ssh"), nil)
		return nil
	case len(args) == 2 && args[0] == "password" && (args[1] == "on" || args[1] == "off"):
		return b.saveSecurity(ctx, l, func(s *settings.Security) error { s.PasswordLoginAlert = args[1] == "on"; return nil })
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
// Successful logins listed in /ssh; the count covers all of them.
const maxLoginLines = 10

func sshSummaryText(ctx context.Context, l i18n.Lang) string {
	st, err := sshwatch.Summary(ctx, 24*time.Hour)
	if err != nil {
		return i18n.T(l, "ssh.unavailable", html.EscapeString(err.Error()))
	}
	var t strings.Builder
	if len(st.Logins) == 0 {
		t.WriteString(i18n.T(l, "ssh.ok_none"))
	} else {
		t.WriteString(i18n.T(l, "ssh.ok_title", sshwatch.FormatCount(len(st.Logins))))
		for i, lg := range st.Logins {
			if i == maxLoginLines {
				t.WriteString("\n  " + i18n.T(l, "ssh.ok_more", len(st.Logins)-maxLoginLines))
				break
			}
			fmt.Fprintf(&t, "\n  <code>%s</code> · <code>%s</code> · %s · %s", html.EscapeString(lg.User),
				html.EscapeString(lg.IP), loginMethod(l, lg), lg.Time.Format("15:04"))
		}
	}
	t.WriteString("\n\n" + i18n.T(l, "ssh.summary", sshwatch.FormatCount(st.Total)))
	if top := sshwatch.FormatTop(l, st); top != "" {
		t.WriteString("\n" + top)
	}
	t.WriteString("\n\n" + sshdConfigText(ctx, l))
	return t.String()
}

func loginMethod(l i18n.Lang, lg sshwatch.Login) string {
	switch {
	case lg.Method == "publickey":
		return i18n.T(l, "ssh.method.key")
	case lg.PasswordLike():
		return i18n.T(l, "ssh.method.password")
	}
	return html.EscapeString(lg.Method)
}

// sshdConfigText shows whether password and root logins are allowed.
func sshdConfigText(ctx context.Context, l i18n.Lang) string {
	c, err := sshwatch.ReadSSHDConfig(ctx)
	if err != nil {
		return i18n.T(l, "ssh.cfg_unknown")
	}
	text := i18n.T(l, "ssh.cfg_title") + "\n"
	if c.PasswordAuth {
		text += i18n.T(l, "ssh.cfg_password_on")
	} else {
		text += i18n.T(l, "ssh.cfg_password_off")
	}
	text += "\n"
	switch c.PermitRoot {
	case "yes":
		text += i18n.T(l, "ssh.cfg_root_yes")
	case "no":
		text += i18n.T(l, "ssh.cfg_root_no")
	default:
		text += i18n.T(l, "ssh.cfg_root_key")
	}
	if c.PasswordAuth || c.PermitRoot == "yes" {
		text += "\n" + i18n.T(l, "ssh.cfg_hint", i18n.DocURL(l, "security.md"))
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
