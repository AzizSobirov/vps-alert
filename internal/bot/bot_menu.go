package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// The menu is one message that is edited in place as the user taps through
// it. Its buttons carry "m:<screen>[:<arg>]".

func init() {
	registerCommand(command{name: "menu", group: "bot", example: "/menu", doc: "telegram.md",
		run: func(b *Bot, ctx context.Context, l i18n.Lang, args []string) error {
			b.showMenu(ctx, l, 0)
			return nil
		}})
	registerCallback("m", (*Bot).menuCallback)
}

// A pendingPrompt is a question from the menu waiting for a typed answer.
// The answer is appended to args and run as the command, so it gets the
// same validation and error messages as typing the command itself.
type pendingPrompt struct {
	msgID int64
	cmd   string
	args  []string
	at    time.Time
}

// Unanswered questions expire, so a stray message hours later is not taken
// as an answer.
const promptTTL = 10 * time.Minute

var (
	retentionPresets = [][3]int{{7, 4, 6}, {14, 8, 12}, {30, 12, 24}, {3, 0, 0}}
	mutePresets      = []string{"30m", "2h", "1d"}
)

func btn(text, data string) telegram.Button { return telegram.Button{Text: text, CallbackData: data} }

// screen edits message msgID, or sends a new message when that is not
// possible. Refreshing an unchanged screen is not an error.
func (b *Bot) screen(ctx context.Context, msgID int64, text string, kb *telegram.Keyboard) {
	if msgID != 0 {
		err := b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, text, kb)
		if err == nil || strings.Contains(err.Error(), "not modified") {
			return
		}
	}
	b.reply(ctx, text, kb)
}

func (b *Bot) showMenu(ctx context.Context, l i18n.Lang, msgID int64) {
	s := b.Store.Get()
	var t strings.Builder
	t.WriteString(i18n.T(l, "menu.title", html.EscapeString(b.Cfg.ServerName)) + "\n")
	switch {
	case s.Backup.LastRun.IsZero():
		t.WriteString(i18n.T(l, "menu.backup_never"))
	case s.Backup.LastOK:
		t.WriteString(i18n.T(l, "menu.backup_ok", s.Backup.LastRun.Format(dateTime)))
	default:
		t.WriteString(i18n.T(l, "menu.backup_failed", s.Backup.LastRun.Format(dateTime)))
	}
	if time.Now().Before(s.MutedUntil) {
		t.WriteString("\n" + i18n.T(l, "menu.muted", s.MutedUntil.Format(dateTime)))
	}
	t.WriteString("\n\n" + i18n.T(l, "menu.pick"))

	kb := &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
		{btn(i18n.T(l, "menu.btn.status"), "m:status"), btn(i18n.T(l, "menu.btn.top"), "m:top")},
		{btn(i18n.T(l, "menu.btn.backup"), "m:backup"), btn(i18n.T(l, "menu.btn.db"), "m:db")},
		{btn(i18n.T(l, "menu.btn.services"), "m:svc"), btn(i18n.T(l, "menu.btn.http"), "m:http")},
		{btn(i18n.T(l, "menu.btn.security"), "m:sec"), btn(i18n.T(l, "menu.btn.alerts"), "m:alerts")},
		{btn(i18n.T(l, "menu.btn.lang"), "m:lang"), btn(i18n.T(l, "menu.btn.help"), "m:help")},
	}}
	b.screen(ctx, msgID, t.String(), kb)
}

func (b *Bot) menuCallback(ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, arg string) {
	msgID := q.Message.MessageID
	screen, rest, _ := strings.Cut(arg, ":")
	toast := ""
	defer func() { b.TG.AnswerCallback(ctx, q.ID, toast) }()

	back := func(to string) []telegram.Button {
		return []telegram.Button{btn(i18n.T(l, "menu.btn.refresh"), "m:"+to), btn(i18n.T(l, "menu.btn.menu"), "m:main")}
	}

	switch screen {
	case "main":
		b.showMenu(ctx, l, msgID)
	case "status":
		b.screen(ctx, msgID, b.statusText(ctx, l), &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{back("status")}})
	case "top":
		text, err := topText(ctx, l)
		if err != nil {
			text = "⚠️ " + html.EscapeString(i18n.Text(l, err))
		}
		b.screen(ctx, msgID, text, &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{back("top")}})
	case "backup":
		b.backupMenu(ctx, msgID)
	case "db":
		b.screen(ctx, msgID, i18n.T(l, "bot.db.scanning"), nil)
		b.scanDatabases(ctx)
		b.showDatabases(ctx, msgID)
	case "svc":
		b.scanServices(ctx)
		b.showServices(ctx, msgID)
	case "http":
		b.httpScreen(ctx, l, msgID)
	case "httprm":
		checks := b.Store.Get().HTTPChecks
		if i, err := strconv.Atoi(rest); err == nil && i >= 0 && i < len(checks) {
			target := checks[i].URL
			b.Store.Update(func(s *settings.Settings) error {
				for j, c := range s.HTTPChecks {
					if c.URL == target {
						s.HTTPChecks = append(s.HTTPChecks[:j], s.HTTPChecks[j+1:]...)
						break
					}
				}
				return nil
			})
			toast = i18n.T(l, "menu.removed")
		}
		b.httpScreen(ctx, l, msgID)
	case "sec":
		b.securityScreen(ctx, l, msgID)
	case "tog":
		b.Store.Update(func(s *settings.Settings) error {
			switch rest {
			case "ssh":
				if s.Security.FailedLoginThreshold > 0 {
					s.Security.FailedLoginThreshold = 0
				} else {
					s.Security.FailedLoginThreshold = defaultFailedLoginThreshold
				}
			case "files":
				s.Security.FileWatch = !s.Security.FileWatch
			case "ports":
				s.Security.PortWatch = !s.Security.PortWatch
			}
			return nil
		})
		toast = i18n.T(l, "bot.saved")
		b.securityScreen(ctx, l, msgID)
	case "ssh":
		b.screen(ctx, msgID, sshSummaryText(ctx, l), &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
			{btn(i18n.T(l, "menu.btn.refresh"), "m:ssh"), btn(i18n.T(l, "menu.btn.back"), "m:sec")}}})
	case "ports":
		b.screen(ctx, msgID, b.portsText(ctx, l), &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
			{btn(i18n.T(l, "menu.btn.refresh"), "m:ports"), btn(i18n.T(l, "menu.btn.back"), "m:sec")}}})
	case "alerts":
		b.alertsScreen(ctx, l, msgID)
	case "mute":
		if d, err := parseDuration(rest); err == nil {
			until := time.Now().Add(d)
			b.Store.Update(func(s *settings.Settings) error { s.MutedUntil = until; return nil })
			toast = i18n.T(l, "bot.saved")
		}
		b.alertsScreen(ctx, l, msgID)
	case "unmute":
		b.Store.Update(func(s *settings.Settings) error { s.MutedUntil = time.Time{}; return nil })
		toast = i18n.T(l, "bot.saved")
		b.alertsScreen(ctx, l, msgID)
	case "ret":
		if rest != "" {
			var n [3]int
			if _, err := fmt.Sscanf(rest, "%d.%d.%d", &n[0], &n[1], &n[2]); err == nil {
				b.Store.Update(func(s *settings.Settings) error {
					s.Backup.KeepDaily, s.Backup.KeepWeekly, s.Backup.KeepMonthly = n[0], n[1], n[2]
					return nil
				})
				toast = i18n.T(l, "bot.saved")
			}
		}
		b.retentionScreen(ctx, l, msgID)
	case "paths":
		b.pathsScreen(ctx, l, msgID)
	case "pathrm":
		paths := b.Store.Get().Backup.Paths
		if i, err := strconv.Atoi(rest); err == nil && i >= 0 && i < len(paths) {
			target := paths[i]
			b.Store.Update(func(s *settings.Settings) error {
				for j, p := range s.Backup.Paths {
					if p == target {
						s.Backup.Paths = append(s.Backup.Paths[:j], s.Backup.Paths[j+1:]...)
						break
					}
				}
				return nil
			})
			toast = i18n.T(l, "menu.removed")
		}
		b.pathsScreen(ctx, l, msgID)
	case "ask":
		b.ask(ctx, l, rest)
	case "lang":
		var row []telegram.Button
		for _, x := range i18n.Langs {
			row = append(row, btn(x.Label(), "lang:"+string(x)))
		}
		b.screen(ctx, msgID, i18n.T(l, "bot.lang.choose"), &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
			row, {btn(i18n.T(l, "menu.btn.menu"), "m:main")}}})
	case "help":
		b.screen(ctx, msgID, helpList(l), &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
			{btn(i18n.T(l, "menu.btn.menu"), "m:main")}}})
	}
}

func (b *Bot) httpScreen(ctx context.Context, l i18n.Lang, msgID int64) {
	checks := b.Store.Get().HTTPChecks
	text := i18n.T(l, "menu.http.title") + "\n\n" + b.httpState(l)
	var rows [][]telegram.Button
	if len(checks) > 0 {
		text += "\n\n" + i18n.T(l, "menu.tap_remove")
	}
	for i, c := range checks {
		rows = append(rows, []telegram.Button{btn("❌ "+shortURL(c.URL), "m:httprm:"+strconv.Itoa(i))})
	}
	rows = append(rows,
		[]telegram.Button{btn(i18n.T(l, "menu.btn.add_url"), "m:ask:url")},
		[]telegram.Button{btn(i18n.T(l, "menu.btn.refresh"), "m:http"), btn(i18n.T(l, "menu.btn.menu"), "m:main")})
	b.screen(ctx, msgID, text, &telegram.Keyboard{InlineKeyboard: rows})
}

func (b *Bot) securityScreen(ctx context.Context, l i18n.Lang, msgID int64) {
	sec := b.Store.Get().Security
	text := i18n.T(l, "menu.sec.title") + "\n\n" + b.sshState(l) + "\n" + b.filesState(l) + "\n" + b.portsState(l)
	kb := &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
		{btn(i18n.T(l, "menu.btn.ssh"), "m:ssh"), btn(i18n.T(l, "menu.btn.ports"), "m:ports")},
		{btn(onOff(sec.FailedLoginThreshold > 0)+" "+i18n.T(l, "menu.btn.tog_ssh"), "m:tog:ssh")},
		{btn(onOff(sec.FileWatch)+" "+i18n.T(l, "menu.btn.tog_files"), "m:tog:files")},
		{btn(onOff(sec.PortWatch)+" "+i18n.T(l, "menu.btn.tog_ports"), "m:tog:ports")},
		{btn(i18n.T(l, "menu.btn.ssh_threshold"), "m:ask:sshthr")},
		{btn(i18n.T(l, "menu.btn.menu"), "m:main")},
	}}
	b.screen(ctx, msgID, text, kb)
}

func (b *Bot) alertsScreen(ctx context.Context, l i18n.Lang, msgID int64) {
	text := i18n.T(l, "menu.alerts.title") + "\n\n" + b.state(l, "alerts") + "\n\n" + b.state(l, "mute") + "\n" + b.state(l, "report")
	var mute []telegram.Button
	for _, d := range mutePresets {
		mute = append(mute, btn("🔕 "+d, "m:mute:"+d))
	}
	mute = append(mute, btn(i18n.T(l, "menu.btn.unmute"), "m:unmute"))
	kb := &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{
		mute,
		{btn("✏️ CPU", "m:ask:alert.cpu"), btn("✏️ RAM", "m:ask:alert.mem"), btn("✏️ "+i18n.T(l, "menu.btn.disk"), "m:ask:alert.disk")},
		{btn(i18n.T(l, "menu.btn.forecast"), "m:ask:alert.forecast"), btn(i18n.T(l, "menu.btn.report_time"), "m:ask:report")},
		{btn(i18n.T(l, "menu.btn.refresh"), "m:alerts"), btn(i18n.T(l, "menu.btn.menu"), "m:main")},
	}}
	b.screen(ctx, msgID, text, kb)
}

func (b *Bot) retentionScreen(ctx context.Context, l i18n.Lang, msgID int64) {
	bk := b.Store.Get().Backup
	text := i18n.T(l, "menu.ret.title") + "\n\n" + b.state(l, "retention") + "\n\n" + i18n.T(l, "menu.ret.pick")
	var rows [][]telegram.Button
	for _, p := range retentionPresets {
		label := i18n.T(l, "menu.ret.preset", p[0], p[1], p[2])
		if p == [3]int{bk.KeepDaily, bk.KeepWeekly, bk.KeepMonthly} {
			label = "✅ " + label
		}
		rows = append(rows, []telegram.Button{btn(label, fmt.Sprintf("m:ret:%d.%d.%d", p[0], p[1], p[2]))})
	}
	rows = append(rows, []telegram.Button{btn(i18n.T(l, "menu.btn.back"), "m:backup")})
	b.screen(ctx, msgID, text, &telegram.Keyboard{InlineKeyboard: rows})
}

func (b *Bot) pathsScreen(ctx context.Context, l i18n.Lang, msgID int64) {
	paths := b.Store.Get().Backup.Paths
	text := i18n.T(l, "menu.paths.title") + "\n\n" + b.state(l, "paths")
	if len(paths) > 0 {
		text += "\n\n" + i18n.T(l, "menu.tap_remove")
	}
	var rows [][]telegram.Button
	for i, p := range paths {
		rows = append(rows, []telegram.Button{btn("❌ "+cut(p, 40), "m:pathrm:"+strconv.Itoa(i))})
	}
	rows = append(rows,
		[]telegram.Button{btn(i18n.T(l, "menu.btn.add_path"), "m:ask:path")},
		[]telegram.Button{btn(i18n.T(l, "menu.btn.back"), "m:backup")})
	b.screen(ctx, msgID, text, &telegram.Keyboard{InlineKeyboard: rows})
}

// ask sends a question whose typed answer completes a command.
func (b *Bot) ask(ctx context.Context, l i18n.Lang, what string) {
	var cmd, key string
	var args []string
	switch {
	case what == "path":
		cmd, args, key = "paths", []string{"add"}, "menu.ask.path"
	case what == "url":
		cmd, args, key = "http", []string{"add"}, "menu.ask.url"
	case what == "time":
		cmd, args, key = "schedule", []string{strconv.Itoa(max(b.Store.Get().Backup.IntervalDays, 1))}, "menu.ask.time"
	case what == "report":
		cmd, key = "report", "menu.ask.report"
	case what == "sshthr":
		cmd, args, key = "ssh", []string{"threshold"}, "menu.ask.ssh_threshold"
	case strings.HasPrefix(what, "alert."):
		cmd, args, key = "alerts", []string{strings.TrimPrefix(what, "alert.")}, "menu.ask.alert"
	default:
		return
	}
	m, err := b.TG.AskReply(ctx, b.Cfg.ChatID, i18n.T(l, key, html.EscapeString(strings.Join(args, " "))))
	if err != nil {
		b.reply(ctx, "⚠️ "+html.EscapeString(err.Error()), nil)
		return
	}
	b.prompt = pendingPrompt{msgID: m.MessageID, cmd: cmd, args: args, at: time.Now()}
}

// isPromptAnswer reports whether m answers the open question: a reply to it,
// or in a private chat the next message unless it is a real command.
func (b *Bot) isPromptAnswer(m *telegram.Message) bool {
	if m.ReplyToMessage != nil && m.ReplyToMessage.MessageID == b.prompt.msgID {
		return true
	}
	if m.Chat.Type != "private" {
		return false
	}
	if cmd, _, ok := parseCommand(m.Text, b.username); ok {
		if _, known := lookupCommand(cmd); known || cmd == "start" {
			return false
		}
	}
	return true
}

// answerPrompt runs the open question's command with m as the answer.
func (b *Bot) answerPrompt(ctx context.Context, m *telegram.Message) {
	p := b.prompt
	if time.Since(p.at) > promptTTL {
		b.prompt = pendingPrompt{}
		return
	}
	answer := strings.TrimSpace(m.Text)
	if answer == "" {
		return
	}
	b.prompt = pendingPrompt{}
	log.Printf("audit: user=%d (@%s) menu answer for /%s", m.From.ID, m.From.Username, p.cmd)
	b.command(ctx, m.From, p.cmd, append(append([]string(nil), p.args...), answer))
}

func onOff(on bool) string {
	if on {
		return "✅"
	}
	return "⬜"
}

// shortURL drops the scheme for button labels.
func shortURL(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return cut(u.Host+strings.TrimSuffix(u.Path, "/"), 40)
	}
	return cut(raw, 40)
}
