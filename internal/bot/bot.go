// Package bot handles Telegram commands. It only takes commands from the
// chat configured in the env file, plus the buttons under alerts in the alert
// chat, and ignores everything else.
package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/backup"
	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/dbdump"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// Commands queued while the agent was offline are dropped if older than this,
// so a "backup now" from yesterday does not run on restart.
const maxCommandAge = 5 * time.Minute

const dateTime = "2006-01-02 15:04"

type Bot struct {
	Cfg    *config.Config
	TG     *telegram.Client
	Store  *settings.Store
	Backup *backup.Manager

	username string

	// Open menu question awaiting a typed answer. Update loop only.
	prompt pendingPrompt

	// Last database scan. Only touched from the update loop, so unlocked.
	found        []settings.Database
	missing      map[settings.Database]bool // selected but not found
	scanWarnings []string

	mu       sync.Mutex
	reported map[int64]time.Time // unauthorized users already reported
}

// command describes a bot command for /help and the Telegram command menu.
// Its texts are the i18n keys "help.<name>.short" and "help.<name>".
type command struct {
	name    string
	group   string // help section, see helpGroups
	example string
	doc     string // page under docs/

	// Commands added with registerCommand carry their own handlers; the
	// built-in ones are dispatched in Bot.command and Bot.state.
	run   func(b *Bot, ctx context.Context, l i18n.Lang, args []string) error
	state func(b *Bot, l i18n.Lang) string // current setting shown above the help, optional
}

// helpGroups orders the sections of /help and the command menu.
var helpGroups = []string{"monitor", "security", "backup", "bot"}

// callbackHandlers handle inline buttons whose data is "<action>:<arg>" for
// actions registered with registerCallback.
var callbackHandlers = map[string]func(b *Bot, ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, arg string){}

// alertCallbacks are the buttons sent under alerts, the only ones that work
// in ALERT_CHAT_ID. Menu buttons edit messages in ALLOWED_CHAT_ID.
var alertCallbacks = map[string]bool{"trust": true}

// registerCommand adds a command from its own file, so features do not all
// have to edit this one. Call it from init.
func registerCommand(c command) {
	if _, dup := lookupCommand(c.name); dup {
		panic("bot: duplicate command " + c.name)
	}
	commands = append(commands, c)
}

// registerCallback adds an inline button handler. Call it from init.
func registerCallback(action string, fn func(b *Bot, ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, arg string)) {
	if _, dup := callbackHandlers[action]; dup {
		panic("bot: duplicate callback " + action)
	}
	callbackHandlers[action] = fn
}

// orderedCommands returns the commands grouped in helpGroups order.
func orderedCommands() []command {
	var out []command
	for _, g := range helpGroups {
		for _, c := range commands {
			if c.group == g {
				out = append(out, c)
			}
		}
	}
	return out
}

var commands = []command{
	{name: "status", group: "monitor", example: "/status", doc: "alerts.md"},
	{name: "alerts", group: "monitor", example: "/alerts cpu 90", doc: "alerts.md"},
	{name: "mute", group: "monitor", example: "/mute 2h", doc: "alerts.md"},
	{name: "unmute", group: "monitor", example: "/unmute", doc: "alerts.md"},
	{name: "report", group: "monitor", example: "/report 09:00", doc: "alerts.md"},
	{name: "backup", group: "backup", example: "/backup", doc: "backups.md"},
	{name: "db", group: "backup", example: "/db", doc: "databases.md"},
	{name: "paths", group: "backup", example: "/paths add /var/www", doc: "backups.md"},
	{name: "schedule", group: "backup", example: "/schedule 1 03:00", doc: "schedule-and-retention.md"},
	{name: "retention", group: "backup", example: "/retention 7 4 6", doc: "schedule-and-retention.md"},
	{name: "lang", group: "bot", example: "/lang uz", doc: "telegram.md"},
	{name: "help", group: "bot", example: "/help schedule", doc: "telegram.md"},
}

func lookupCommand(name string) (command, bool) {
	for _, c := range commands {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

// usageError marks an error caused by wrong arguments; the command's help is
// shown after it.
type usageError struct{ error }

func (e usageError) Unwrap() error { return e.error }

func usage(err error) error { return usageError{err} }

func (b *Bot) Run(ctx context.Context) {
	for {
		me, err := b.TG.GetMe(ctx)
		if err == nil {
			b.username = me.Username
			break
		}
		log.Printf("bot: getMe: %v", err)
		if !sleep(ctx, 10*time.Second) {
			return
		}
	}
	if err := b.TG.SetMyCommands(ctx, botCommands(b.language())); err != nil {
		log.Printf("bot: setMyCommands: %v", err)
	}
	log.Printf("bot: polling as @%s", b.username)

	var offset int64
	for {
		ups, err := b.TG.GetUpdates(ctx, offset, 50)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			wait := 5 * time.Second
			var re *telegram.RetryError
			if errors.As(err, &re) {
				wait = re.After
			}
			log.Printf("bot: getUpdates: %v", err)
			if !sleep(ctx, wait) {
				return
			}
			continue
		}
		for _, u := range ups {
			offset = u.UpdateID + 1
			b.handle(ctx, u)
		}
	}
}

// language is the bot's current language: the /lang setting, else LANGUAGE
// from the env file, else English.
func (b *Bot) language() i18n.Lang {
	return i18n.Pick(b.Store.Get().Language, b.Cfg.Language)
}

func botCommands(l i18n.Lang) []telegram.BotCommand {
	var cmds []telegram.BotCommand
	for _, c := range orderedCommands() {
		cmds = append(cmds, telegram.BotCommand{Command: c.name, Description: i18n.T(l, "help."+c.name+".short")})
	}
	return cmds
}

func (b *Bot) handle(ctx context.Context, u telegram.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("bot: panic handling update %d: %v", u.UpdateID, r)
		}
	}()

	switch {
	case u.MyChatMember != nil:
		// Leaving a chat produces an update too; only react to being added.
		m := u.MyChatMember
		if status := m.NewChatMember.Status; !b.ownChat(m.Chat.ID) && status != "left" && status != "kicked" {
			b.reject(ctx, m.Chat, m.From, i18n.T(b.language(), "bot.unauthorized.added"))
		}
	case u.Message != nil:
		m := u.Message
		// The alert chat only receives; commands there are not taken.
		if m.From == nil || (m.Chat.ID != b.Cfg.ChatID && m.Chat.ID == b.Cfg.AlertChatID) {
			return
		}
		// An answer to a menu question may itself start with "/", like a
		// path, so it is checked before command parsing.
		if b.prompt.msgID != 0 && b.allowed(m.Chat.ID, m.From.ID) && b.isPromptAnswer(m) {
			b.answerPrompt(ctx, m)
			return
		}
		if !strings.HasPrefix(m.Text, "/") {
			return
		}
		if !b.allowed(m.Chat.ID, m.From.ID) {
			b.reject(ctx, m.Chat, *m.From, m.Text)
			return
		}
		if time.Since(time.Unix(m.Date, 0)) > maxCommandAge {
			return
		}
		cmd, args, ok := parseCommand(m.Text, b.username)
		if !ok {
			return
		}
		log.Printf("audit: user=%d (@%s) command=%q", m.From.ID, m.From.Username, m.Text)
		b.command(ctx, m.From, cmd, args)
	case u.CallbackQuery != nil:
		q := u.CallbackQuery
		if q.Message == nil || !b.buttonAllowed(q.Message.Chat.ID, q.From.ID, q.Data) {
			b.TG.AnswerCallback(ctx, q.ID, i18n.T(b.language(), "bot.not_allowed"))
			if q.Message != nil {
				b.reject(ctx, q.Message.Chat, q.From, "button "+q.Data)
			}
			return
		}
		log.Printf("audit: user=%d (@%s) button=%q", q.From.ID, q.From.Username, q.Data)
		b.callback(ctx, q)
	}
}

func (b *Bot) allowed(chatID, userID int64) bool {
	if chatID != b.Cfg.ChatID {
		return false
	}
	return len(b.Cfg.UserIDs) == 0 || b.Cfg.UserIDs[userID]
}

// buttonAllowed also takes alert buttons pressed in ALERT_CHAT_ID. That can be
// a channel anyone may read, so there only ALLOWED_USER_IDS count, or with
// none set, the owner of a private ALLOWED_CHAT_ID.
func (b *Bot) buttonAllowed(chatID, userID int64, data string) bool {
	if chatID == b.Cfg.ChatID {
		return b.allowed(chatID, userID)
	}
	action, _, _ := strings.Cut(data, ":")
	if chatID != b.Cfg.AlertChatID || !alertCallbacks[action] {
		return false
	}
	if len(b.Cfg.UserIDs) > 0 {
		return b.Cfg.UserIDs[userID]
	}
	return userID == b.Cfg.ChatID
}

// ownChat reports whether the bot belongs in chatID.
func (b *Bot) ownChat(chatID int64) bool {
	return chatID == b.Cfg.ChatID || chatID == b.Cfg.AlertChatID
}

// reject leaves foreign group chats and tells the owner about the attempt,
// at most once per user per hour.
func (b *Bot) reject(ctx context.Context, chat telegram.Chat, from telegram.User, what string) {
	log.Printf("security: rejected user=%d (@%s) chat=%d: %q", from.ID, from.Username, chat.ID, what)
	if !b.ownChat(chat.ID) && chat.Type != "private" {
		if err := b.TG.LeaveChat(ctx, chat.ID); err != nil {
			log.Printf("bot: leaveChat %d: %v", chat.ID, err)
		}
	}

	b.mu.Lock()
	if b.reported == nil {
		b.reported = map[int64]time.Time{}
	}
	last := b.reported[from.ID]
	recent := time.Since(last) < time.Hour
	if !recent {
		b.reported[from.ID] = time.Now()
	}
	b.mu.Unlock()
	if recent {
		return
	}

	if len(what) > 100 {
		what = what[:100] + "…"
	}
	b.reply(ctx, i18n.T(b.language(), "bot.unauthorized",
		html.EscapeString(displayName(from)), from.ID, chat.ID, html.EscapeString(chat.Type), html.EscapeString(what)), nil)
}

// parseCommand splits "/cmd@bot arg1 arg2". Commands addressed to another
// bot are ignored.
func parseCommand(text, username string) (string, []string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return "", nil, false
	}
	cmd := strings.TrimPrefix(fields[0], "/")
	if name, target, found := strings.Cut(cmd, "@"); found {
		if !strings.EqualFold(target, username) {
			return "", nil, false
		}
		cmd = name
	}
	return strings.ToLower(cmd), fields[1:], true
}

func (b *Bot) command(ctx context.Context, from *telegram.User, cmd string, args []string) {
	l := b.language()
	var err error
	switch cmd {
	case "start":
		b.showMenu(ctx, l, 0)
	case "help":
		err = b.help(ctx, l, args)
	case "status":
		b.status(ctx, l)
	case "backup":
		b.backupMenu(ctx, 0)
	case "db":
		m, err := b.TG.SendMessage(ctx, b.Cfg.ChatID, i18n.T(l, "bot.db.scanning"), nil)
		if err == nil {
			b.scanDatabases(ctx)
			b.showDatabases(ctx, m.MessageID)
		}
	case "paths":
		err = b.paths(ctx, l, args)
	case "schedule":
		err = b.schedule(ctx, l, args)
	case "retention":
		err = b.retention(ctx, l, args)
	case "alerts":
		err = b.alerts(ctx, l, args)
	case "report":
		err = b.report(ctx, l, args)
	case "mute":
		err = b.mute(ctx, l, args)
	case "unmute":
		err = b.Store.Update(func(s *settings.Settings) error { s.MutedUntil = time.Time{}; return nil })
		if err == nil {
			b.reply(ctx, i18n.T(l, "bot.unmuted"), nil)
		}
	case "lang":
		err = b.lang(ctx, l, args)
	default:
		if c, ok := lookupCommand(cmd); ok && c.run != nil {
			err = c.run(b, ctx, l, args)
		} else {
			b.reply(ctx, i18n.T(l, "bot.unknown"), nil)
		}
	}
	if err != nil {
		msg := "⚠️ " + html.EscapeString(i18n.Text(l, err))
		if errors.As(err, new(usageError)) {
			msg += "\n\n" + commandHelp(l, cmd)
		}
		b.reply(ctx, msg, nil)
	}
}

func (b *Bot) callback(ctx context.Context, q *telegram.CallbackQuery) {
	l := b.language()
	msgID := q.Message.MessageID
	action, arg, _ := strings.Cut(q.Data, ":")
	switch action {
	case "now":
		if b.Backup.Running() {
			b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.backup.already_running"))
			return
		}
		b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.backup.started_toast"))
		b.reply(ctx, i18n.T(l, "bot.backup.started"), nil)
		reason := i18n.T(l, "bot.backup.reason_manual", displayName(q.From))
		go func() {
			if err := b.Backup.Run(ctx, reason); errors.Is(err, backup.ErrRunning) {
				b.reply(ctx, "⚠️ "+i18n.T(l, "bot.backup.already_running"), nil)
			}
		}()
	case "snapshots":
		b.TG.AnswerCallback(ctx, q.ID, "")
		b.snapshots(ctx, l)
	case "target":
		if arg == "" {
			b.TG.AnswerCallback(ctx, q.ID, "")
			b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, i18n.T(l, "bot.target.title"), &telegram.Keyboard{
				InlineKeyboard: [][]telegram.Button{{
					{Text: i18n.T(l, "bot.target.local"), CallbackData: "target:local"},
					{Text: i18n.T(l, "bot.target.r2"), CallbackData: "target:r2"},
					{Text: i18n.T(l, "bot.target.both"), CallbackData: "target:both"},
				}, {{Text: i18n.T(l, "bot.btn.back"), CallbackData: "menu"}}},
			})
			return
		}
		if arg != "local" && !b.Cfg.R2Enabled() {
			b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.target.r2_missing"))
			return
		}
		b.update(ctx, l, q, func(s *settings.Settings) error { s.Backup.Target = arg; return nil })
		b.backupMenu(ctx, msgID)
	case "every":
		if arg == "" {
			b.TG.AnswerCallback(ctx, q.ID, "")
			b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, i18n.T(l, "bot.every.title"), &telegram.Keyboard{
				InlineKeyboard: [][]telegram.Button{{
					{Text: i18n.T(l, "bot.every.daily"), CallbackData: "every:1"},
					{Text: i18n.T(l, "bot.every.3days"), CallbackData: "every:3"},
					{Text: i18n.T(l, "bot.every.weekly"), CallbackData: "every:7"},
				}, {
					{Text: i18n.T(l, "menu.btn.time"), CallbackData: "m:ask:time"},
					{Text: i18n.T(l, "bot.every.off"), CallbackData: "every:off"},
					{Text: i18n.T(l, "bot.btn.back"), CallbackData: "menu"},
				}},
			})
			return
		}
		b.update(ctx, l, q, func(s *settings.Settings) error {
			if arg == "off" {
				s.Backup.Enabled = false
				return nil
			}
			days, _ := strconv.Atoi(arg)
			s.Backup.Enabled, s.Backup.IntervalDays = true, days
			s.Backup.NextRun = settings.NextOccurrence(time.Now(), s.Backup.Time)
			return nil
		})
		b.backupMenu(ctx, msgID)
	case "dbscan":
		b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.db.scanning_toast"))
		b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, i18n.T(l, "bot.db.scanning"), nil)
		b.scanDatabases(ctx)
		b.showDatabases(ctx, msgID)
	case "db":
		i, err := strconv.Atoi(arg)
		if err != nil || i < 0 || i >= len(b.found) {
			b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.db.outdated"))
			b.scanDatabases(ctx)
			b.showDatabases(ctx, msgID)
			return
		}
		db := b.found[i]
		b.update(ctx, l, q, func(s *settings.Settings) error {
			if idx := indexOf(s.Backup.Databases, db); idx >= 0 {
				s.Backup.Databases = append(s.Backup.Databases[:idx], s.Backup.Databases[idx+1:]...)
			} else {
				s.Backup.Databases = append(s.Backup.Databases, db)
			}
			return nil
		})
		b.showDatabases(ctx, msgID)
	case "lang":
		next, ok := i18n.Parse(arg)
		if !ok {
			b.TG.AnswerCallback(ctx, q.ID, "")
			return
		}
		if err := b.setLanguage(ctx, next); err != nil {
			b.TG.AnswerCallback(ctx, q.ID, i18n.Text(l, err))
			return
		}
		b.TG.AnswerCallback(ctx, q.ID, i18n.T(next, "bot.saved"))
	case "menu":
		b.TG.AnswerCallback(ctx, q.ID, "")
		b.backupMenu(ctx, msgID)
	default:
		if fn, ok := callbackHandlers[action]; ok {
			fn(b, ctx, l, q, arg)
		} else {
			b.TG.AnswerCallback(ctx, q.ID, "")
		}
	}
}

func (b *Bot) update(ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, fn func(*settings.Settings) error) {
	if err := b.Store.Update(fn); err != nil {
		b.TG.AnswerCallback(ctx, q.ID, i18n.Text(l, err))
		return
	}
	b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.saved"))
}

func (b *Bot) status(ctx context.Context, l i18n.Lang) {
	b.reply(ctx, b.statusText(ctx, l), nil)
}

func (b *Bot) statusText(ctx context.Context, l i18n.Lang) string {
	snap, err := metrics.Collect(ctx)
	if err != nil {
		return "⚠️ " + html.EscapeString(err.Error())
	}
	text := snap.Format(l, b.Cfg.ServerName)
	s := b.Store.Get()
	if time.Now().Before(s.MutedUntil) {
		text += "\n" + i18n.T(l, "bot.status.muted", s.MutedUntil.Format(dateTime))
	}
	if b.Backup.Running() {
		text += "\n" + i18n.T(l, "bot.status.backup_running")
	}
	return text
}

// backupMenu sends the backup overview, or edits message msgID in place.
func (b *Bot) backupMenu(ctx context.Context, msgID int64) {
	l := b.language()
	s := b.Store.Get().Backup
	var t strings.Builder
	t.WriteString(i18n.T(l, "bot.backup.title") + "\n\n")
	if s.Enabled {
		t.WriteString(i18n.T(l, "bot.backup.schedule_on", s.IntervalDays, s.Time, s.NextRun.Format(dateTime)) + "\n")
	} else {
		t.WriteString(i18n.T(l, "bot.backup.schedule_off") + "\n")
	}
	t.WriteString(i18n.T(l, "bot.backup.target", i18n.T(l, "bot.target."+s.Target)) + "\n")
	t.WriteString(i18n.T(l, "bot.backup.keep", s.KeepDaily, s.KeepWeekly, s.KeepMonthly) + "\n")
	if len(s.Paths) == 0 {
		t.WriteString(i18n.T(l, "bot.backup.paths_none") + "\n")
	} else {
		t.WriteString(i18n.T(l, "bot.backup.paths") + "\n")
		for _, p := range s.Paths {
			fmt.Fprintf(&t, "  • <code>%s</code>\n", html.EscapeString(p))
		}
	}
	if len(s.Databases) == 0 {
		t.WriteString(i18n.T(l, "bot.backup.dbs_none") + "\n")
	} else {
		t.WriteString(i18n.T(l, "bot.backup.dbs") + "\n")
		for _, d := range s.Databases {
			fmt.Fprintf(&t, "  • %s\n", html.EscapeString(d.Label()))
		}
	}
	switch {
	case s.LastRun.IsZero():
		t.WriteString(i18n.T(l, "bot.backup.last_never") + "\n")
	case s.LastOK:
		t.WriteString(i18n.T(l, "bot.backup.last_ok", s.LastRun.Format(dateTime)) + "\n")
	default:
		t.WriteString(i18n.T(l, "bot.backup.last_failed", s.LastRun.Format(dateTime)) + "\n")
	}
	if err := b.Backup.Available(); err != nil {
		fmt.Fprintf(&t, "\n⚠️ %s\n", html.EscapeString(i18n.Text(l, err)))
	}

	kb := &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{{
		{Text: i18n.T(l, "bot.btn.backup_now"), CallbackData: "now"},
		{Text: i18n.T(l, "bot.btn.snapshots"), CallbackData: "snapshots"},
	}, {
		{Text: i18n.T(l, "bot.btn.target"), CallbackData: "target"},
		{Text: i18n.T(l, "bot.btn.schedule"), CallbackData: "every"},
	}, {
		{Text: i18n.T(l, "menu.btn.retention"), CallbackData: "m:ret"},
		{Text: i18n.T(l, "menu.btn.paths"), CallbackData: "m:paths"},
	}, {
		{Text: i18n.T(l, "bot.btn.databases"), CallbackData: "dbscan"},
		{Text: i18n.T(l, "menu.btn.menu"), CallbackData: "m:main"},
	}}}
	if msgID != 0 {
		if err := b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, t.String(), kb); err == nil {
			return
		}
	}
	b.reply(ctx, t.String(), kb)
}

// maxDatabaseButtons keeps the keyboard well under Telegram's limits.
const maxDatabaseButtons = 40

// scanDatabases refreshes the list shown by showDatabases. Selected
// databases that were not found stay listed, so they can be deselected.
func (b *Bot) scanDatabases(ctx context.Context) {
	found, warnings := dbdump.Discover(ctx)
	b.missing = map[settings.Database]bool{}
	for _, db := range b.Store.Get().Backup.Databases {
		if indexOf(found, db) < 0 {
			found = append(found, db)
			b.missing[db] = true
		}
	}
	if len(found) > maxDatabaseButtons {
		found = found[:maxDatabaseButtons]
	}
	b.found, b.scanWarnings = found, warnings
}

func (b *Bot) showDatabases(ctx context.Context, msgID int64) {
	l := b.language()
	selected := b.Store.Get().Backup.Databases
	var t strings.Builder
	t.WriteString(i18n.T(l, "bot.db.title") + "\n")
	if len(b.found) == 0 {
		t.WriteString("\n" + i18n.T(l, "bot.db.none") + "\n")
	} else {
		t.WriteString(i18n.T(l, "bot.db.hint") + "\n")
	}
	for _, w := range b.scanWarnings {
		fmt.Fprintf(&t, "\n⚠️ <code>%s</code>", html.EscapeString(w))
	}

	var rows [][]telegram.Button
	for i, db := range b.found {
		mark := "⬜"
		if indexOf(selected, db) >= 0 {
			mark = "✅"
		}
		label := mark + " " + db.Label()
		if b.missing[db] {
			label += " " + i18n.T(l, "bot.db.not_found")
		}
		rows = append(rows, []telegram.Button{{Text: label, CallbackData: "db:" + strconv.Itoa(i)}})
	}
	rows = append(rows, []telegram.Button{
		{Text: i18n.T(l, "bot.btn.rescan"), CallbackData: "dbscan"},
		{Text: i18n.T(l, "bot.btn.back"), CallbackData: "menu"},
	})
	kb := &telegram.Keyboard{InlineKeyboard: rows}
	if err := b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, t.String(), kb); err != nil {
		b.reply(ctx, t.String(), kb)
	}
}

func indexOf(dbs []settings.Database, db settings.Database) int {
	for i, d := range dbs {
		if d == db {
			return i
		}
	}
	return -1
}

func (b *Bot) snapshots(ctx context.Context, l i18n.Lang) {
	var t strings.Builder
	t.WriteString(i18n.T(l, "bot.snapshots.title") + "\n")
	for _, target := range backup.Targets(b.Store.Get().Backup.Target) {
		fmt.Fprintf(&t, "\n<b>%s</b>\n", i18n.T(l, "bot.target."+target))
		snaps, err := b.Backup.Snapshots(ctx, target, 10)
		switch {
		case err != nil:
			fmt.Fprintf(&t, "⚠️ <code>%s</code>\n", html.EscapeString(i18n.Text(l, err)))
		case len(snaps) == 0:
			t.WriteString(i18n.T(l, "bot.snapshots.none") + "\n")
		}
		for _, s := range snaps {
			size := ""
			if s.Summary != nil {
				size = ", " + metrics.Bytes(s.Summary.TotalBytesProcessed)
			}
			fmt.Fprintf(&t, "<code>%s</code> %s%s\n", s.ShortID, s.Time.Local().Format(dateTime), size)
		}
	}
	b.reply(ctx, t.String(), nil)
}

func (b *Bot) paths(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "paths")
		return nil
	}
	if len(args) != 2 || (args[0] != "add" && args[0] != "rm") {
		return usage(i18n.Err("bot.err.usage"))
	}
	p := filepath.Clean(args[1])
	if !filepath.IsAbs(p) {
		return usage(i18n.Err("bot.err.path_absolute"))
	}
	err := b.Store.Update(func(s *settings.Settings) error {
		idx := -1
		for i, x := range s.Backup.Paths {
			if x == p {
				idx = i
			}
		}
		if args[0] == "rm" {
			if idx < 0 {
				return i18n.Err("bot.err.path_not_listed", p)
			}
			s.Backup.Paths = append(s.Backup.Paths[:idx], s.Backup.Paths[idx+1:]...)
			return nil
		}
		if idx >= 0 {
			return i18n.Err("bot.err.path_listed", p)
		}
		if _, err := os.Stat(p); err != nil {
			return i18n.Err("bot.err.path_missing", p)
		}
		s.Backup.Paths = append(s.Backup.Paths, p)
		return nil
	})
	if err != nil {
		return err
	}
	b.reply(ctx, i18n.T(l, "bot.saved_msg"), nil)
	b.backupMenu(ctx, 0)
	return nil
}

func (b *Bot) schedule(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "schedule")
		return nil
	}
	var msg string
	err := b.Store.Update(func(s *settings.Settings) error {
		if len(args) == 1 && args[0] == "off" {
			s.Backup.Enabled = false
			msg = i18n.T(l, "bot.schedule.off")
			return nil
		}
		if len(args) != 2 {
			return usage(i18n.Err("bot.err.usage"))
		}
		days, err := strconv.Atoi(args[0])
		if err != nil || days < 1 || days > 365 {
			return usage(i18n.Err("bot.err.days"))
		}
		if _, _, err := settings.ParseClock(args[1]); err != nil {
			return usage(i18n.Err("bot.err.clock", args[1]))
		}
		s.Backup.Enabled, s.Backup.IntervalDays, s.Backup.Time = true, days, args[1]
		s.Backup.NextRun = settings.NextOccurrence(time.Now(), args[1])
		msg = i18n.T(l, "bot.schedule.saved", days, args[1], s.Backup.NextRun.Format(dateTime))
		return nil
	})
	if err == nil {
		b.reply(ctx, msg, nil)
	}
	return err
}

func (b *Bot) retention(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "retention")
		return nil
	}
	if len(args) != 3 {
		return usage(i18n.Err("bot.err.usage"))
	}
	var n [3]int
	for i, a := range args {
		v, err := strconv.Atoi(a)
		if err != nil || v < 0 || v > 1000 {
			return usage(i18n.Err("bot.err.number", a))
		}
		n[i] = v
	}
	if n[0]+n[1]+n[2] == 0 {
		return usage(i18n.Err("bot.err.keep_one"))
	}
	err := b.Store.Update(func(s *settings.Settings) error {
		s.Backup.KeepDaily, s.Backup.KeepWeekly, s.Backup.KeepMonthly = n[0], n[1], n[2]
		return nil
	})
	if err == nil {
		b.reply(ctx, i18n.T(l, "bot.retention.saved", n[0], n[1], n[2]), nil)
	}
	return err
}

func (b *Bot) alerts(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "alerts")
		return nil
	}
	if len(args) != 2 {
		return usage(i18n.Err("bot.err.usage"))
	}
	v, err := strconv.Atoi(args[1])
	if err != nil {
		return usage(i18n.Err("bot.err.number", args[1]))
	}
	err = b.Store.Update(func(s *settings.Settings) error {
		a := &s.Alerts
		if pctKey := map[string]bool{"cpu": true, "mem": true, "disk": true, "inode": true}[args[0]]; pctKey && (v < 0 || v > 100) {
			return usage(i18n.Err("bot.err.percent"))
		}
		switch args[0] {
		case "cpu":
			a.CPUPercent = float64(v)
		case "mem":
			a.MemPercent = float64(v)
		case "disk":
			a.DiskPercent = float64(v)
		case "inode":
			a.InodePercent = float64(v)
		case "duration":
			if v < 0 || v > 1440 {
				return usage(i18n.Err("bot.err.duration"))
			}
			a.DurationMin = v
		case "cooldown":
			if v < 1 || v > 1440 {
				return usage(i18n.Err("bot.err.cooldown"))
			}
			a.CooldownMin = v
		case "forecast":
			if v < 0 || v > 90 {
				return usage(i18n.Err("bot.err.forecast"))
			}
			a.ForecastDays = v
		default:
			return usage(i18n.Err("bot.err.alert_key", args[0]))
		}
		return nil
	})
	if err == nil {
		b.reply(ctx, i18n.T(l, "bot.saved_msg"), nil)
	}
	return err
}

func (b *Bot) report(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "report")
		return nil
	}
	if len(args) != 1 {
		return usage(i18n.Err("bot.err.usage"))
	}
	var msg string
	err := b.Store.Update(func(s *settings.Settings) error {
		if args[0] == "off" {
			s.Report.Enabled = false
			msg = i18n.T(l, "bot.report.off")
			return nil
		}
		if _, _, err := settings.ParseClock(args[0]); err != nil {
			return usage(i18n.Err("bot.err.clock", args[0]))
		}
		s.Report.Enabled, s.Report.Time = true, args[0]
		// Do not send a second report today just because the time moved.
		if !time.Now().Before(settings.At(time.Now(), args[0])) {
			s.Report.LastSent = time.Now().Format("2006-01-02")
		}
		msg = i18n.T(l, "bot.report.saved", args[0])
		return nil
	})
	if err == nil {
		b.reply(ctx, msg, nil)
	}
	return err
}

func (b *Bot) mute(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "mute")
		return nil
	}
	if len(args) != 1 {
		return usage(i18n.Err("bot.err.usage"))
	}
	d, err := parseDuration(args[0])
	if err != nil || d <= 0 || d > 7*24*time.Hour {
		return usage(i18n.Err("bot.err.mute_duration"))
	}
	until := time.Now().Add(d)
	if err := b.Store.Update(func(s *settings.Settings) error { s.MutedUntil = until; return nil }); err != nil {
		return err
	}
	b.reply(ctx, i18n.T(l, "bot.mute.saved", until.Format(dateTime)), nil)
	return nil
}

func (b *Bot) lang(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		var row []telegram.Button
		for _, x := range i18n.Langs {
			row = append(row, telegram.Button{Text: x.Label(), CallbackData: "lang:" + string(x)})
		}
		b.reply(ctx, i18n.T(l, "bot.lang.choose"), &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{row}})
		return nil
	}
	next, ok := i18n.Parse(args[0])
	if !ok || len(args) != 1 {
		return usage(i18n.Err("bot.err.lang", args[0]))
	}
	return b.setLanguage(ctx, next)
}

// setLanguage saves the language, translates the Telegram command menu and
// confirms in the new language.
func (b *Bot) setLanguage(ctx context.Context, l i18n.Lang) error {
	if err := b.Store.Update(func(s *settings.Settings) error { s.Language = string(l); return nil }); err != nil {
		return err
	}
	if err := b.TG.SetMyCommands(ctx, botCommands(l)); err != nil {
		log.Printf("bot: setMyCommands: %v", err)
	}
	b.reply(ctx, i18n.T(l, "bot.lang.saved"), nil)
	return nil
}

func (b *Bot) help(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.reply(ctx, helpList(l), nil)
		return nil
	}
	name := strings.ToLower(strings.TrimPrefix(args[0], "/"))
	if _, ok := lookupCommand(name); !ok {
		return usage(i18n.Err("bot.help.unknown", args[0]))
	}
	b.showHelp(ctx, l, name)
	return nil
}

// showHelp sends a command's current state, if it has one, followed by its
// detailed help.
func (b *Bot) showHelp(ctx context.Context, l i18n.Lang, name string) {
	text := commandHelp(l, name)
	if state := b.state(l, name); state != "" {
		text = state + "\n\n" + text
	}
	b.reply(ctx, text, nil)
}

// state describes the current setting a command changes.
func (b *Bot) state(l i18n.Lang, name string) string {
	s := b.Store.Get()
	switch name {
	case "schedule":
		line := i18n.T(l, "bot.state.schedule_off")
		if s.Backup.Enabled {
			line = i18n.T(l, "bot.state.schedule_on", s.Backup.IntervalDays, s.Backup.Time, s.Backup.NextRun.Format(dateTime))
		}
		return line + "\n" + i18n.T(l, "bot.state.clock", time.Now().Format("15:04 MST"))
	case "retention":
		return i18n.T(l, "bot.state.retention", s.Backup.KeepDaily, s.Backup.KeepWeekly, s.Backup.KeepMonthly)
	case "report":
		if !s.Report.Enabled {
			return i18n.T(l, "bot.state.report_off")
		}
		return i18n.T(l, "bot.state.report_on", s.Report.Time)
	case "mute":
		if time.Now().Before(s.MutedUntil) {
			return i18n.T(l, "bot.state.muted", s.MutedUntil.Format(dateTime))
		}
		return i18n.T(l, "bot.state.unmuted")
	case "paths":
		if len(s.Backup.Paths) == 0 {
			return i18n.T(l, "bot.state.paths_none")
		}
		var t strings.Builder
		t.WriteString(i18n.T(l, "bot.state.paths"))
		for _, p := range s.Backup.Paths {
			fmt.Fprintf(&t, "\n  • <code>%s</code>", html.EscapeString(p))
		}
		return t.String()
	case "alerts":
		a := s.Alerts
		forecast := i18n.T(l, "bot.state.forecast_off")
		if a.ForecastDays > 0 {
			forecast = i18n.T(l, "bot.state.forecast", a.ForecastDays)
		}
		return i18n.T(l, "bot.state.alerts", pct(l, a.CPUPercent), pct(l, a.MemPercent), pct(l, a.DiskPercent),
			pct(l, a.InodePercent), a.DurationMin, a.CooldownMin) + "\n" + forecast
	}
	if c, ok := lookupCommand(name); ok && c.state != nil {
		return c.state(b, l)
	}
	return ""
}

// commandHelp is the detailed help for one command: what it does, usage,
// examples and a link to the documentation.
func commandHelp(l i18n.Lang, name string) string {
	c, _ := lookupCommand(name)
	return fmt.Sprintf("<b>/%s</b>: %s\n\n%s\n\n%s", name, html.EscapeString(i18n.T(l, "help."+name+".short")),
		i18n.T(l, "help."+name), i18n.T(l, "bot.help.more", i18n.DocURL(l, c.doc)))
}

// helpList lists every command with a one-line description and an example.
func helpList(l i18n.Lang) string {
	var t strings.Builder
	t.WriteString(i18n.T(l, "bot.help.title") + "\n")
	group := ""
	for _, c := range orderedCommands() {
		if c.group != group {
			group = c.group
			fmt.Fprintf(&t, "\n%s\n", i18n.T(l, "bot.help.group."+group))
		}
		fmt.Fprintf(&t, "/%s: %s\n", c.name, html.EscapeString(i18n.T(l, "help."+c.name+".short")))
		if c.example != "/"+c.name {
			fmt.Fprintf(&t, "    <code>%s</code>\n", html.EscapeString(c.example))
		}
	}
	t.WriteString("\n" + i18n.T(l, "bot.help.footer"))
	return t.String()
}

func (b *Bot) reply(ctx context.Context, text string, kb *telegram.Keyboard) {
	b.send(ctx, b.Cfg.ChatID, text, kb)
}

func (b *Bot) send(ctx context.Context, chatID int64, text string, kb *telegram.Keyboard) {
	if _, err := b.TG.SendMessage(ctx, chatID, text, kb); err != nil {
		log.Printf("bot: send: %v", err)
	}
}

func parseDuration(s string) (time.Duration, error) {
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		return time.Duration(n) * 24 * time.Hour, err
	}
	return time.ParseDuration(s)
}

func pct(l i18n.Lang, v float64) string {
	if v <= 0 {
		return i18n.T(l, "bot.off")
	}
	return fmt.Sprintf("%.0f%%", v)
}

func displayName(u telegram.User) string {
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.FirstName
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
