package bot

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/httpcheck"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/metrics"
	"github.com/AzizSobirov/vps-alert/internal/settings"
)

func init() {
	registerCommand(command{
		name: "http", group: "monitor", example: "/http add https://example.com", doc: "services.md",
		run:   (*Bot).httpChecks,
		state: (*Bot).httpState,
	})
}

func (b *Bot) httpChecks(ctx context.Context, l i18n.Lang, args []string) error {
	if len(args) == 0 {
		b.showHelp(ctx, l, "http")
		return nil
	}
	if len(args) != 2 || (args[0] != "add" && args[0] != "rm") {
		return usage(i18n.Err("http.err.usage"))
	}
	u := args[1]
	if args[0] == "add" {
		if err := httpcheck.Validate(u); err != nil {
			return usage(err)
		}
	}
	err := b.Store.Update(func(s *settings.Settings) error {
		idx := -1
		for i, c := range s.HTTPChecks {
			if c.URL == u {
				idx = i
			}
		}
		if args[0] == "rm" {
			if idx < 0 {
				return i18n.Err("http.err.not_found", u)
			}
			s.HTTPChecks = append(s.HTTPChecks[:idx], s.HTTPChecks[idx+1:]...)
			return nil
		}
		if idx >= 0 {
			return i18n.Err("http.err.duplicate", u)
		}
		if len(s.HTTPChecks) >= httpcheck.MaxChecks {
			return i18n.Err("http.err.max", httpcheck.MaxChecks)
		}
		s.HTTPChecks = append(s.HTTPChecks, settings.HTTPCheck{URL: u})
		return nil
	})
	if err != nil {
		return err
	}
	key := "http.bot.added"
	if args[0] == "rm" {
		key = "http.bot.removed"
	}
	b.reply(ctx, i18n.T(l, key, html.EscapeString(u)), nil)
	return nil
}

// httpState lists the checked URLs with their latest result.
func (b *Bot) httpState(l i18n.Lang) string {
	checks := b.Store.Get().HTTPChecks
	if len(checks) == 0 {
		return i18n.T(l, "http.bot.state_none")
	}
	var t strings.Builder
	t.WriteString(i18n.T(l, "http.bot.state"))
	for _, c := range checks {
		u := html.EscapeString(c.URL)
		st, ok := httpcheck.Snapshot(c.URL)
		switch {
		case !ok:
			fmt.Fprintf(&t, "\n⚪ %s %s", u, i18n.T(l, "http.bot.pending"))
		case st.Up:
			fmt.Fprintf(&t, "\n🟢 %s (%d ms)", u, st.Latency.Milliseconds())
		default:
			line := fmt.Sprintf("\n🔴 %s: %s", u, html.EscapeString(httpcheck.Problem(l, st.Kind, st.Detail)))
			if !st.DownSince.IsZero() {
				line += " " + i18n.T(l, "http.bot.down_for", metrics.FormatDuration(time.Since(st.DownSince)))
			}
			t.WriteString(line)
		}
	}
	return t.String()
}
