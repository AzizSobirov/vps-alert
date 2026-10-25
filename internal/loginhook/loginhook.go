// Package loginhook reports logins. It is run by pam_exec on every session
// open, e.g. from /etc/pam.d/sshd:
//
//	session optional pam_exec.so quiet /usr/local/bin/vps-alert pam-hook
package loginhook

import (
	"context"
	"encoding/json"
	"html"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/knownips"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
	"github.com/AzizSobirov/vps-alert/internal/trust"
)

const childEnv = "VPS_ALERT_PAM_CHILD"

// Detach re-executes the current command in a new session and returns
// immediately, so a slow Telegram API never delays the login itself.
// It reports whether the caller is the detached child and should continue.
func Detach() (child bool, err error) {
	if os.Getenv(childEnv) != "" {
		return true, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = append(os.Environ(), childEnv+"=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return false, cmd.Start()
}

type Event struct {
	User    string
	Host    string // remote host, empty for local logins
	Service string
	TTY     string
}

// FromEnv reads the event from pam_exec's environment. ok is false for PAM
// stages other than session open.
func FromEnv() (ev Event, ok bool) {
	if os.Getenv("PAM_TYPE") != "open_session" {
		return ev, false
	}
	return Event{
		User:    os.Getenv("PAM_USER"),
		Host:    os.Getenv("PAM_RHOST"),
		Service: os.Getenv("PAM_SERVICE"),
		TTY:     os.Getenv("PAM_TTY"),
	}, true
}

func Whitelisted(host string, nets []*net.IPNet) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func Notify(cfg *config.Config, ev Event) error {
	text, kb, silent := Message(cfg, savedSettings(cfg.SettingsPath()), ev, time.Now())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := telegram.New(cfg.BotToken).SendMessageOpts(ctx, cfg.ChatID, text, kb, silent)
	return err
}

// Message builds the login alert. A trusted login (whitelisted address or
// user, from the env file or /whitelist) is sent silently but still sent. An
// untrusted one from an IP address gets a button to trust that address, and
// is marked when the address has never logged in before.
func Message(cfg *config.Config, s settings.Settings, ev Event, now time.Time) (text string, kb *telegram.Keyboard, silent bool) {
	l := i18n.Pick(s.Language, cfg.Language)
	trusted := (ev.Host != "" && Whitelisted(ev.Host, cfg.LoginWhitelist)) ||
		trust.Match(s.Security.LoginWhitelist, ev.User, ev.Host)

	from := ev.Host
	if from == "" {
		from = i18n.T(l, "login.local")
	}
	text = i18n.T(l, "login.alert",
		html.EscapeString(cfg.ServerName), html.EscapeString(ev.User), html.EscapeString(from),
		html.EscapeString(ev.Service), now.Format("2006-01-02 15:04:05 MST"))
	if trusted {
		return text + "\n" + i18n.T(l, "login.trusted"), nil, true
	}
	if ev.Host != "" && NewAddress(filepath.Join(cfg.DataDir, knownips.FileName), ev.Host, now) {
		text = i18n.T(l, "login.new_ip") + "\n" + text
	}
	if net.ParseIP(ev.Host) != nil {
		kb = &telegram.Keyboard{InlineKeyboard: [][]telegram.Button{{
			{Text: i18n.T(l, "login.btn.trust"), CallbackData: "trust:" + ev.Host},
		}}}
	}
	return text, kb, false
}

// NewAddress reports whether host has never logged in before, and records
// it. Until the service has created the list (it seeds it from the logs on
// first start) nothing is flagged, so a fresh install does not call every
// address new.
func NewAddress(path, host string, now time.Time) bool {
	known, exists := knownips.Load(path)
	if !exists {
		return false
	}
	_, seen := known[host]
	knownips.Add(path, now, host)
	return !seen
}

// savedSettings reads the settings changed from the bot (language,
// whitelist). It only reads the file: the PAM hook must never create it.
func savedSettings(path string) settings.Settings {
	var s settings.Settings
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &s)
	}
	return s
}
