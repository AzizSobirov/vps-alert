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
	"syscall"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
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
	if ev.Host != "" && Whitelisted(ev.Host, cfg.LoginWhitelist) {
		return nil
	}
	l := i18n.Pick(savedLanguage(cfg.SettingsPath()), cfg.Language)
	from := ev.Host
	if from == "" {
		from = i18n.T(l, "login.local")
	}
	text := i18n.T(l, "login.alert",
		html.EscapeString(cfg.ServerName), html.EscapeString(ev.User), html.EscapeString(from),
		html.EscapeString(ev.Service), time.Now().Format("2006-01-02 15:04:05 MST"))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := telegram.New(cfg.BotToken).SendMessage(ctx, cfg.ChatID, text, nil)
	return err
}

// savedLanguage reads the language chosen with /lang. It only reads the
// settings file: the PAM hook must never create files.
func savedLanguage(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var s settings.Settings
	if json.Unmarshal(data, &s) != nil {
		return ""
	}
	return s.Language
}
