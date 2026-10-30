package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
	"github.com/AzizSobirov/vps-alert/internal/trust"
)

// maxWhitelist keeps the list, and its menu screen, a reasonable size.
const maxWhitelist = 50

func init() {
	registerCommand(command{name: "whitelist", group: "security", example: "/whitelist add deploy", doc: "security.md",
		run: (*Bot).whitelistCommand, state: (*Bot).whitelistState})
	// The "It was me" button under a login alert, sent by the PAM hook.
	registerCallback("trust", func(b *Bot, ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, arg string) {
		entry, err := b.addWhitelist(arg)
		if err != nil && !errors.Is(err, errWhitelistExists) {
			b.TG.AnswerCallback(ctx, q.ID, i18n.Text(l, err))
			return
		}
		b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "bot.saved"))
		// The alert may be in ALERT_CHAT_ID; answer where it was.
		chat := q.Message.Chat.ID
		b.TG.EditReplyMarkup(ctx, chat, q.Message.MessageID, nil)
		b.send(ctx, chat, i18n.T(l, "whitelist.trusted_by", html.EscapeString(entry), html.EscapeString(displayName(q.From))), nil)
	})
}

var errWhitelistExists = i18n.Err("whitelist.err.exists")

func (b *Bot) whitelistCommand(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "whitelist")
		return nil
	}
	if len(args) != 2 || (args[0] != "add" && args[0] != "rm") {
		return usage(i18n.Err("bot.err.usage"))
	}
	if args[0] == "add" {
		entry, err := b.addWhitelist(args[1])
		if err != nil {
			return err
		}
		b.reply(ctx, i18n.T(l, "whitelist.added", html.EscapeString(entry)), nil)
		return nil
	}
	entry, err := b.removeWhitelist(args[1])
	if err != nil {
		return err
	}
	b.reply(ctx, i18n.T(l, "whitelist.removed", html.EscapeString(entry)), nil)
	return nil
}

func (b *Bot) addWhitelist(raw string) (string, error) {
	entry, err := trust.Normalize(raw)
	if err != nil {
		return "", usage(i18n.Err("whitelist.err.invalid", raw))
	}
	err = b.Store.Update(func(s *settings.Settings) error {
		list := s.Security.LoginWhitelist
		for _, e := range list {
			if e == entry {
				return errWhitelistExists
			}
		}
		if len(list) >= maxWhitelist {
			return i18n.Err("whitelist.err.max", maxWhitelist)
		}
		s.Security.LoginWhitelist = append(list, entry)
		return nil
	})
	return entry, err
}

func (b *Bot) removeWhitelist(raw string) (string, error) {
	entry, err := trust.Normalize(raw)
	if err != nil {
		entry = strings.TrimSpace(raw)
	}
	err = b.Store.Update(func(s *settings.Settings) error {
		for i, e := range s.Security.LoginWhitelist {
			if e == entry {
				s.Security.LoginWhitelist = append(s.Security.LoginWhitelist[:i], s.Security.LoginWhitelist[i+1:]...)
				return nil
			}
		}
		return i18n.Err("whitelist.err.not_found", entry)
	})
	return entry, err
}

// whitelistState lists the entries added from the bot and those set in the
// env file, which can only be changed on the server.
func (b *Bot) whitelistState(l i18n.Lang) string {
	list := b.Store.Get().Security.LoginWhitelist
	var t strings.Builder
	if len(list) == 0 {
		t.WriteString(i18n.T(l, "whitelist.none"))
	} else {
		t.WriteString(i18n.T(l, "whitelist.title"))
		for _, e := range list {
			fmt.Fprintf(&t, "\n  %s", whitelistLine(l, e))
		}
	}
	if len(b.Cfg.LoginWhitelist) > 0 {
		t.WriteString("\n" + i18n.T(l, "whitelist.env"))
		for _, n := range b.Cfg.LoginWhitelist {
			fmt.Fprintf(&t, "\n  🌐 <code>%s</code>", html.EscapeString(n.String()))
		}
	}
	return t.String()
}

func whitelistLine(l i18n.Lang, entry string) string {
	if trust.IsUser(entry) {
		return i18n.T(l, "whitelist.user", html.EscapeString(entry))
	}
	return "🌐 <code>" + html.EscapeString(entry) + "</code>"
}
