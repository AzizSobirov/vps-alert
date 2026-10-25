package bot

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/services"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

const maxServiceButtons = 40

// Last /services scan. Only touched from the update loop, so unlocked.
var (
	svcFound  []services.Item
	svcStates map[services.Item]services.State
)

func init() {
	registerCommand(command{
		name: "services", group: "monitor", example: "/services", doc: "services.md",
		run: func(b *Bot, ctx context.Context, l i18n.Lang, args []string) error {
			m, err := b.TG.SendMessage(ctx, b.Cfg.ChatID, i18n.T(l, "services.bot.scanning"), nil)
			if err != nil {
				return err
			}
			b.scanServices(ctx)
			b.showServices(ctx, m.MessageID)
			return nil
		},
		state: func(b *Bot, l i18n.Lang) string {
			items := services.Selected(b.Store.Get().Services)
			if len(items) == 0 {
				return i18n.T(l, "services.bot.state_none")
			}
			var t strings.Builder
			t.WriteString(i18n.T(l, "services.bot.state"))
			for _, it := range items {
				fmt.Fprintf(&t, "\n  • %s", html.EscapeString(it.Label()))
			}
			return t.String()
		},
	})
	registerCallback("svcscan", func(b *Bot, ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, arg string) {
		b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "services.bot.scanning_toast"))
		b.TG.EditMessage(ctx, b.Cfg.ChatID, q.Message.MessageID, i18n.T(l, "services.bot.scanning"), nil)
		b.scanServices(ctx)
		b.showServices(ctx, q.Message.MessageID)
	})
	registerCallback("svc", func(b *Bot, ctx context.Context, l i18n.Lang, q *telegram.CallbackQuery, arg string) {
		i, err := strconv.Atoi(arg)
		if err != nil || i < 0 || i >= len(svcFound) {
			b.TG.AnswerCallback(ctx, q.ID, i18n.T(l, "services.bot.outdated"))
			b.scanServices(ctx)
			b.showServices(ctx, q.Message.MessageID)
			return
		}
		it := svcFound[i]
		b.update(ctx, l, q, func(s *settings.Settings) error {
			list := &s.Services.Units
			if it.Kind == services.Docker {
				list = &s.Services.Containers
			}
			if idx := svcIndex(*list, it.Name); idx >= 0 {
				*list = append((*list)[:idx], (*list)[idx+1:]...)
			} else {
				*list = append(*list, it.Name)
			}
			return nil
		})
		b.showServices(ctx, q.Message.MessageID)
	})
}

// scanServices lists running services and containers plus the selected
// ones, which stay listed even when stopped so they can be deselected.
func (b *Bot) scanServices(ctx context.Context) {
	found := services.Discover(ctx)
	for _, it := range services.Selected(b.Store.Get().Services) {
		if !svcContains(found, it) {
			found = append(found, it)
		}
	}
	if len(found) > maxServiceButtons {
		found = found[:maxServiceButtons]
	}
	svcStates = map[services.Item]services.State{}
	for _, it := range found {
		svcStates[it] = services.Check(ctx, it)
	}
	svcFound = found
}

func (b *Bot) showServices(ctx context.Context, msgID int64) {
	l := b.language()
	selected := services.Selected(b.Store.Get().Services)
	var t strings.Builder
	t.WriteString(i18n.T(l, "services.bot.title") + "\n")
	if len(svcFound) == 0 {
		t.WriteString("\n" + i18n.T(l, "services.bot.none"))
	} else {
		t.WriteString(i18n.T(l, "services.bot.hint"))
	}

	var rows [][]telegram.Button
	for i, it := range svcFound {
		mark := "⬜"
		if svcContains(selected, it) {
			mark = "✅"
		}
		st := svcStates[it]
		dot := "🟢"
		if !st.OK {
			dot = "🔴"
		}
		label := mark + " " + dot + " " + it.Label()
		if st.Missing {
			label += " " + i18n.T(l, "services.bot.not_found")
		} else if !st.OK && st.Status != "" {
			label += " (" + services.StatusText(l, st.Status) + ")"
		}
		rows = append(rows, []telegram.Button{{Text: label, CallbackData: "svc:" + strconv.Itoa(i)}})
	}
	rows = append(rows, []telegram.Button{
		{Text: i18n.T(l, "bot.btn.rescan"), CallbackData: "svcscan"},
		{Text: i18n.T(l, "menu.btn.menu"), CallbackData: "m:main"},
	})
	kb := &telegram.Keyboard{InlineKeyboard: rows}
	if err := b.TG.EditMessage(ctx, b.Cfg.ChatID, msgID, t.String(), kb); err != nil {
		b.reply(ctx, t.String(), kb)
	}
}

func svcContains(items []services.Item, it services.Item) bool {
	for _, x := range items {
		if x == it {
			return true
		}
	}
	return false
}

func svcIndex(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}
