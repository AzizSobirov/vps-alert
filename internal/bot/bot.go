// Package bot handles Telegram commands. It only ever talks to the single
// chat configured in the env file and ignores everything else.
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
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// Commands queued while the agent was offline are dropped if older than this,
// so a "backup now" from yesterday does not run on restart.
const maxCommandAge = 5 * time.Minute

type Bot struct {
	Cfg    *config.Config
	TG     *telegram.Client
	Store  *settings.Store
	Backup *backup.Manager

	username string

	// Last database scan. Only touched from the update loop, so unlocked.
	found        []settings.Database
	missing      map[settings.Database]bool // selected but not found
	scanWarnings []string

	mu       sync.Mutex
	reported map[int64]time.Time // unauthorized users already reported
}

var commands = []telegram.BotCommand{
	{Command: "status", Description: "Server status"},
	{Command: "backup", Description: "Backup menu"},
	{Command: "db", Description: "Databases to back up"},
	{Command: "paths", Description: "Backup paths: /paths add|rm /path"},
	{Command: "schedule", Description: "Backup schedule: /schedule <days> <HH:MM> | off"},
	{Command: "retention", Description: "Keep snapshots: /retention <daily> <weekly> <monthly>"},
	{Command: "alerts", Description: "Alert thresholds: /alerts cpu 90"},
	{Command: "report", Description: "Daily report time: /report 09:00 | off"},
	{Command: "mute", Description: "Mute alerts: /mute 1h"},
	{Command: "unmute", Description: "Unmute alerts"},
	{Command: "help", Description: "Help"},
}

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
	if err := b.TG.SetMyCommands(ctx, commands); err != nil {
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
		if status := m.NewChatMember.Status; m.Chat.ID != b.Cfg.ChatID && status != "left" && status != "kicked" {
			b.reject(ctx, m.Chat, m.From, "added the bot to a chat")
		}
	case u.Message != nil:
		m := u.Message
		if m.From == nil || !strings.HasPrefix(m.Text, "/") {
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
		if q.Message == nil || !b.allowed(q.Message.Chat.ID, q.From.ID) {
			b.TG.AnswerCallback(ctx, q.ID, "Not allowed")
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

// reject leaves foreign group chats and tells the owner about the attempt,
// at most once per user per hour.
func (b *Bot) reject(ctx context.Context, chat telegram.Chat, from telegram.User, what string) {
	log.Printf("security: rejected user=%d (@%s) chat=%d: %q", from.ID, from.Username, chat.ID, what)
	if chat.ID != b.Cfg.ChatID && chat.Type != "private" {
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
	b.reply(ctx, fmt.Sprintf("⚠️ <b>Unauthorized access attempt</b>\nUser: %s (<code>%d</code>)\nChat: <code>%d</code> (%s)\nAction: <code>%s</code>",
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
	var err error
	switch cmd {
	case "start", "help":
		b.reply(ctx, helpText, nil)
	case "status":
		b.status(ctx)
	case "backup":
		b.backupMenu(ctx, 0)
	case "db":
		m, err := b.TG.SendMessage(ctx, b.Cfg.ChatID, "🔍 Scanning for databases…", nil)
		if err == nil {
			b.scanDatabases(ctx)
			b.showDatabases(ctx, m.MessageID)
		}
	case "paths":
		err = b.paths(ctx, args)
	case "schedule":
		err = b.schedule(ctx, args)
	case "retention":
		err = b.retention(ctx, args)
	case "alerts":
		err = b.alerts(ctx, args)
	case "report":
		err = b.report(ctx, args)
	case "mute":
		err = b.mute(ctx, args)
	case "unmute":
		err = b.Store.Update(func(s *settings.Settings) error { s.MutedUntil = time.Time{}; return nil })
		if err == nil {
			b.reply(ctx, "🔔 Alerts unmuted", nil)
		}
	default:
		b.reply(ctx, "Unknown command. /help", nil)
	}
	if err != nil {
		b.reply(ctx, "⚠️ "+html.EscapeString(err.Error()), nil)
	}
}

func (b *Bot) callback(ctx context.Context, q *telegram.CallbackQuery) {
	msgID := q.Message.MessageID
	action, arg, _ := strings.Cut(q.Data, ":")
	switch action {
	case "now":
		if b.Backup.Running() {
			b.TG.AnswerCallback(ctx, q.ID, "A backup is already running")
			return
		}
		b.TG.AnswerCallback(ctx, q.ID, "Backup started")
		b.reply(ctx, "⏳ Backup started…", nil)
		reason := "manual, " + displayName(q.From)
		go func() {
			if err := b.Backup.Run(ctx, reason); errors.Is(err, backup.ErrRunning) {
				b.reply(ctx, "⚠️ A backup is already running", nil)
			}
		}()
	case "snapshots":
		b.TG.AnswerCallback(ctx, q.ID, "")
		b.snapshots(ctx)
	case "target":
		if arg == "" {
			b.TG.AnswerCallback(ctx, q.ID, "")
			b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, "🎯 <b>Where should backups go?</b>", &telegram.Keyboard{
				InlineKeyboard: [][]telegram.Button{{
					{Text: "💽 Local", CallbackData: "target:local"},
					{Text: "☁️ R2", CallbackData: "target:r2"},
					{Text: "Both", CallbackData: "target:both"},
				}, {{Text: "« Back", CallbackData: "menu"}}},
			})
			return
		}
		if arg != "local" && !b.Cfg.R2Enabled() {
			b.TG.AnswerCallback(ctx, q.ID, "R2 is not configured in the env file")
			return
		}
		b.update(ctx, q, func(s *settings.Settings) error { s.Backup.Target = arg; return nil })
		b.backupMenu(ctx, msgID)
	case "every":
		if arg == "" {
			b.TG.AnswerCallback(ctx, q.ID, "")
			b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, "📅 <b>How often?</b>\nFor a custom time use /schedule &lt;days&gt; &lt;HH:MM&gt;", &telegram.Keyboard{
				InlineKeyboard: [][]telegram.Button{{
					{Text: "Daily", CallbackData: "every:1"},
					{Text: "Every 3 days", CallbackData: "every:3"},
					{Text: "Weekly", CallbackData: "every:7"},
				}, {
					{Text: "⏸ Off", CallbackData: "every:off"},
					{Text: "« Back", CallbackData: "menu"},
				}},
			})
			return
		}
		b.update(ctx, q, func(s *settings.Settings) error {
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
		b.TG.AnswerCallback(ctx, q.ID, "Scanning…")
		b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, "🔍 Scanning for databases…", nil)
		b.scanDatabases(ctx)
		b.showDatabases(ctx, msgID)
	case "db":
		i, err := strconv.Atoi(arg)
		if err != nil || i < 0 || i >= len(b.found) {
			b.TG.AnswerCallback(ctx, q.ID, "List is outdated, scanning again")
			b.scanDatabases(ctx)
			b.showDatabases(ctx, msgID)
			return
		}
		db := b.found[i]
		b.update(ctx, q, func(s *settings.Settings) error {
			if idx := indexOf(s.Backup.Databases, db); idx >= 0 {
				s.Backup.Databases = append(s.Backup.Databases[:idx], s.Backup.Databases[idx+1:]...)
			} else {
				s.Backup.Databases = append(s.Backup.Databases, db)
			}
			return nil
		})
		b.showDatabases(ctx, msgID)
	case "menu":
		b.TG.AnswerCallback(ctx, q.ID, "")
		b.backupMenu(ctx, msgID)
	default:
		b.TG.AnswerCallback(ctx, q.ID, "")
	}
}

func (b *Bot) update(ctx context.Context, q *telegram.CallbackQuery, fn func(*settings.Settings) error) {
	if err := b.Store.Update(fn); err != nil {
		b.TG.AnswerCallback(ctx, q.ID, err.Error())
		return
	}
	b.TG.AnswerCallback(ctx, q.ID, "Saved")
}

func (b *Bot) status(ctx context.Context) {
	snap, err := metrics.Collect(ctx)
	if err != nil {
		b.reply(ctx, "⚠️ "+html.EscapeString(err.Error()), nil)
		return
	}
	text := snap.Format(b.Cfg.ServerName)
	s := b.Store.Get()
	if time.Now().Before(s.MutedUntil) {
		text += fmt.Sprintf("\n🔕 Alerts muted until %s", s.MutedUntil.Format("2006-01-02 15:04"))
	}
	if b.Backup.Running() {
		text += "\n⏳ Backup in progress"
	}
	b.reply(ctx, text, nil)
}

// backupMenu sends the backup overview, or edits message msgID in place.
func (b *Bot) backupMenu(ctx context.Context, msgID int64) {
	s := b.Store.Get().Backup
	var t strings.Builder
	t.WriteString("💾 <b>Backup</b>\n\n")
	if s.Enabled {
		fmt.Fprintf(&t, "📅 Every %d day(s) at %s\n⏭ Next: %s\n", s.IntervalDays, s.Time, s.NextRun.Format("2006-01-02 15:04"))
	} else {
		t.WriteString("📅 Schedule: off\n")
	}
	fmt.Fprintf(&t, "🎯 Target: %s\n", s.Target)
	fmt.Fprintf(&t, "🗂 Keep: %d daily, %d weekly, %d monthly\n", s.KeepDaily, s.KeepWeekly, s.KeepMonthly)
	if len(s.Paths) == 0 {
		t.WriteString("📁 Paths: <i>none, add with /paths add /var/www</i>\n")
	} else {
		t.WriteString("📁 Paths:\n")
		for _, p := range s.Paths {
			fmt.Fprintf(&t, "  • <code>%s</code>\n", html.EscapeString(p))
		}
	}
	if len(s.Databases) == 0 {
		t.WriteString("🗄 Databases: <i>none, choose with /db</i>\n")
	} else {
		t.WriteString("🗄 Databases:\n")
		for _, d := range s.Databases {
			fmt.Fprintf(&t, "  • %s\n", html.EscapeString(d.Label()))
		}
	}
	switch {
	case s.LastRun.IsZero():
		t.WriteString("🕓 Last run: never\n")
	case s.LastOK:
		fmt.Fprintf(&t, "🕓 Last run: ✅ %s\n", s.LastRun.Format("2006-01-02 15:04"))
	default:
		fmt.Fprintf(&t, "🕓 Last run: ❌ %s\n", s.LastRun.Format("2006-01-02 15:04"))
	}
	if err := b.Backup.Available(); err != nil {
		fmt.Fprintf(&t, "\n⚠️ %s\n", html.EscapeString(err.Error()))
	}

	kb := &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{{
		{Text: "▶️ Backup now", CallbackData: "now"},
		{Text: "📋 Snapshots", CallbackData: "snapshots"},
	}, {
		{Text: "🎯 Target", CallbackData: "target"},
		{Text: "📅 Schedule", CallbackData: "every"},
	}, {
		{Text: "🗄 Databases", CallbackData: "dbscan"},
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
	selected := b.Store.Get().Backup.Databases
	var t strings.Builder
	t.WriteString("🗄 <b>Databases</b>\n")
	if len(b.found) == 0 {
		t.WriteString("\nNo databases found. Supported: PostgreSQL and MySQL/MariaDB, on the host or in Docker.\n")
	} else {
		t.WriteString("Tap to include or exclude from backups. Each one is dumped to its own file.\n")
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
			label += " (not found)"
		}
		rows = append(rows, []telegram.Button{{Text: label, CallbackData: "db:" + strconv.Itoa(i)}})
	}
	rows = append(rows, []telegram.Button{
		{Text: "🔄 Rescan", CallbackData: "dbscan"},
		{Text: "« Back", CallbackData: "menu"},
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

func (b *Bot) snapshots(ctx context.Context) {
	var t strings.Builder
	t.WriteString("📋 <b>Recent snapshots</b>\n")
	for _, target := range backup.Targets(b.Store.Get().Backup.Target) {
		fmt.Fprintf(&t, "\n<b>%s</b>\n", target)
		snaps, err := b.Backup.Snapshots(ctx, target, 10)
		switch {
		case err != nil:
			fmt.Fprintf(&t, "⚠️ <code>%s</code>\n", html.EscapeString(err.Error()))
		case len(snaps) == 0:
			t.WriteString("<i>no snapshots yet</i>\n")
		}
		for _, s := range snaps {
			size := ""
			if s.Summary != nil {
				size = ", " + metrics.Bytes(s.Summary.TotalBytesProcessed)
			}
			fmt.Fprintf(&t, "<code>%s</code> %s%s\n", s.ShortID, s.Time.Local().Format("2006-01-02 15:04"), size)
		}
	}
	b.reply(ctx, t.String(), nil)
}

func (b *Bot) paths(ctx context.Context, args []string) error {
	if len(args) == 0 {
		b.backupMenu(ctx, 0)
		return nil
	}
	if len(args) != 2 || (args[0] != "add" && args[0] != "rm") {
		return errors.New("usage: /paths add /path or /paths rm /path")
	}
	p := filepath.Clean(args[1])
	if !filepath.IsAbs(p) {
		return errors.New("path must be absolute")
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
				return fmt.Errorf("%s is not in the list", p)
			}
			s.Backup.Paths = append(s.Backup.Paths[:idx], s.Backup.Paths[idx+1:]...)
			return nil
		}
		if idx >= 0 {
			return fmt.Errorf("%s is already in the list", p)
		}
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("%s does not exist on the server", p)
		}
		s.Backup.Paths = append(s.Backup.Paths, p)
		return nil
	})
	if err != nil {
		return err
	}
	b.reply(ctx, "✅ Saved", nil)
	b.backupMenu(ctx, 0)
	return nil
}

func (b *Bot) schedule(ctx context.Context, args []string) error {
	var msg string
	err := b.Store.Update(func(s *settings.Settings) error {
		if len(args) == 1 && args[0] == "off" {
			s.Backup.Enabled = false
			msg = "⏸ Scheduled backups turned off"
			return nil
		}
		if len(args) != 2 {
			return errors.New("usage: /schedule <days> <HH:MM>, e.g. /schedule 1 03:00, or /schedule off")
		}
		days, err := strconv.Atoi(args[0])
		if err != nil || days < 1 || days > 365 {
			return errors.New("days must be between 1 and 365")
		}
		if _, _, err := settings.ParseClock(args[1]); err != nil {
			return err
		}
		s.Backup.Enabled, s.Backup.IntervalDays, s.Backup.Time = true, days, args[1]
		s.Backup.NextRun = settings.NextOccurrence(time.Now(), args[1])
		msg = fmt.Sprintf("✅ Backup every %d day(s) at %s. Next: %s", days, args[1], s.Backup.NextRun.Format("2006-01-02 15:04"))
		return nil
	})
	if err == nil {
		b.reply(ctx, msg, nil)
	}
	return err
}

func (b *Bot) retention(ctx context.Context, args []string) error {
	if len(args) != 3 {
		return errors.New("usage: /retention <daily> <weekly> <monthly>, e.g. /retention 7 4 6")
	}
	var n [3]int
	for i, a := range args {
		v, err := strconv.Atoi(a)
		if err != nil || v < 0 || v > 1000 {
			return fmt.Errorf("invalid number %q", a)
		}
		n[i] = v
	}
	if n[0]+n[1]+n[2] == 0 {
		return errors.New("keep at least one snapshot")
	}
	err := b.Store.Update(func(s *settings.Settings) error {
		s.Backup.KeepDaily, s.Backup.KeepWeekly, s.Backup.KeepMonthly = n[0], n[1], n[2]
		return nil
	})
	if err == nil {
		b.reply(ctx, fmt.Sprintf("✅ Keeping %d daily, %d weekly, %d monthly snapshots", n[0], n[1], n[2]), nil)
	}
	return err
}

func (b *Bot) alerts(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a := b.Store.Get().Alerts
		b.reply(ctx, fmt.Sprintf("🚨 <b>Alert thresholds</b>\n\nCPU: %s\nRAM: %s\nDisk: %s\nInodes: %s\nMust last: %d min\nRepeat every: %d min\n\nChange: <code>/alerts cpu 90</code> (0 disables)\nKeys: cpu, mem, disk, inode, duration, cooldown",
			pct(a.CPUPercent), pct(a.MemPercent), pct(a.DiskPercent), pct(a.InodePercent), a.DurationMin, a.CooldownMin), nil)
		return nil
	}
	if len(args) != 2 {
		return errors.New("usage: /alerts <key> <value>, e.g. /alerts disk 85")
	}
	v, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid number %q", args[1])
	}
	err = b.Store.Update(func(s *settings.Settings) error {
		a := &s.Alerts
		if pctKey := map[string]bool{"cpu": true, "mem": true, "disk": true, "inode": true}[args[0]]; pctKey && (v < 0 || v > 100) {
			return errors.New("percent must be between 0 and 100")
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
				return errors.New("duration must be between 0 and 1440 minutes")
			}
			a.DurationMin = v
		case "cooldown":
			if v < 1 || v > 1440 {
				return errors.New("cooldown must be between 1 and 1440 minutes")
			}
			a.CooldownMin = v
		default:
			return fmt.Errorf("unknown key %q", args[0])
		}
		return nil
	})
	if err == nil {
		b.reply(ctx, "✅ Saved", nil)
	}
	return err
}

func (b *Bot) report(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: /report 09:00 or /report off")
	}
	var msg string
	err := b.Store.Update(func(s *settings.Settings) error {
		if args[0] == "off" {
			s.Report.Enabled = false
			msg = "⏸ Daily report turned off"
			return nil
		}
		if _, _, err := settings.ParseClock(args[0]); err != nil {
			return err
		}
		s.Report.Enabled, s.Report.Time = true, args[0]
		// Do not send a second report today just because the time moved.
		if !time.Now().Before(settings.At(time.Now(), args[0])) {
			s.Report.LastSent = time.Now().Format("2006-01-02")
		}
		msg = "✅ Daily report at " + args[0]
		return nil
	})
	if err == nil {
		b.reply(ctx, msg, nil)
	}
	return err
}

func (b *Bot) mute(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: /mute 30m, /mute 2h or /mute 1d")
	}
	d, err := parseDuration(args[0])
	if err != nil || d <= 0 || d > 7*24*time.Hour {
		return errors.New("duration must be between 1m and 7d, e.g. 30m, 2h, 1d")
	}
	until := time.Now().Add(d)
	if err := b.Store.Update(func(s *settings.Settings) error { s.MutedUntil = until; return nil }); err != nil {
		return err
	}
	b.reply(ctx, "🔕 Alerts muted until "+until.Format("2006-01-02 15:04"), nil)
	return nil
}

func (b *Bot) reply(ctx context.Context, text string, kb *telegram.Keyboard) {
	if _, err := b.TG.SendMessage(ctx, b.Cfg.ChatID, text, kb); err != nil {
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

func pct(v float64) string {
	if v <= 0 {
		return "off"
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

const helpText = `🛡 <b>vps-alert</b>: server monitoring and backups

<b>Status</b>
/status: CPU, RAM, disk, uptime
/alerts: show or change thresholds
/mute 1h, /unmute: silence alerts
/report 09:00: daily report time

<b>Backup</b>
/backup: menu (run now, snapshots, target, schedule)
/db: find databases and choose which to back up
/paths add /var/www: add a path
/paths rm /var/www: remove a path
/schedule 1 03:00: every N days at HH:MM
/retention 7 4 6: snapshots to keep (daily, weekly, monthly)`
