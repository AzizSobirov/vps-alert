package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AzizSobirov/vps-alert/internal/backup"
	"github.com/AzizSobirov/vps-alert/internal/bot"
	"github.com/AzizSobirov/vps-alert/internal/config"
	"github.com/AzizSobirov/vps-alert/internal/events"
	"github.com/AzizSobirov/vps-alert/internal/filewatch"
	"github.com/AzizSobirov/vps-alert/internal/httpcheck"
	"github.com/AzizSobirov/vps-alert/internal/i18n"
	"github.com/AzizSobirov/vps-alert/internal/loginhook"
	"github.com/AzizSobirov/vps-alert/internal/monitor"
	"github.com/AzizSobirov/vps-alert/internal/restore"
	"github.com/AzizSobirov/vps-alert/internal/services"
	"github.com/AzizSobirov/vps-alert/internal/settings"
	"github.com/AzizSobirov/vps-alert/internal/sshwatch"
	"github.com/AzizSobirov/vps-alert/internal/telegram"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `vps-alert: server monitoring, login alerts and backups via Telegram

Usage:
  vps-alert [-env FILE] [command]

Commands:
  run        start the agent (default)
  test       send a test message to the configured chat
  restore    restore files or a database from a backup (interactive)
  pam-hook   report a login (called by pam_exec)
  version    print the version
`

func main() {
	envFile := flag.String("env", "/etc/vps-alert/vps-alert.env", "env file with configuration")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage); flag.PrintDefaults() }
	flag.Parse()

	// journald adds its own timestamps.
	if os.Getenv("INVOCATION_ID") != "" {
		log.SetFlags(0)
	}

	cmd := flag.Arg(0)
	if cmd == "" {
		cmd = "run"
	}

	if cmd == "pam-hook" {
		pamHook(*envFile)
		return
	}
	if cmd == "version" {
		fmt.Println(version)
		return
	}

	if err := loadEnv(*envFile); err != nil {
		log.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	switch cmd {
	case "run":
		if err := run(cfg); err != nil {
			log.Fatal(err)
		}
	case "restore":
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		store, err := settings.Open(cfg.SettingsPath())
		if err != nil {
			log.Fatal(err)
		}
		bm := &backup.Manager{Cfg: cfg, Store: store}
		if err := restore.Run(ctx, bm, &telegram.Notifier{TG: telegram.New(cfg.BotToken), ChatID: cfg.AlertChatID}); err != nil {
			log.Fatal(err)
		}
	case "test":
		chats := []int64{cfg.ChatID}
		if cfg.AlertChatID != cfg.ChatID {
			chats = append(chats, cfg.AlertChatID)
		}
		tg := telegram.New(cfg.BotToken)
		for _, chat := range chats {
			_, err := tg.SendMessage(context.Background(), chat,
				i18n.T(i18n.Pick(cfg.Language), "main.test", html.EscapeString(cfg.ServerName)), nil)
			if err != nil {
				log.Fatalf("chat %d: %v", chat, err)
			}
			fmt.Printf("Test message sent to %d.\n", chat)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// loadEnv loads the env file. A missing default file is fine (the variables
// may come from systemd or the shell); a missing explicit one is not.
func loadEnv(path string) error {
	explicit := false
	flag.Visit(func(f *flag.Flag) { explicit = explicit || f.Name == "env" })
	err := config.LoadEnvFile(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return nil
	}
	return err
}

func run(cfg *config.Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return err
	}
	store, err := settings.Open(cfg.SettingsPath())
	if err != nil {
		return err
	}
	events.SetDir(cfg.DataDir)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tg := telegram.New(cfg.BotToken)
	notify := &telegram.Notifier{TG: tg, ChatID: cfg.AlertChatID}
	backups := &backup.Manager{Cfg: cfg, Store: store, Notify: notify}

	if len(cfg.UserIDs) == 0 && cfg.ChatID < 0 {
		log.Printf("warning: ALLOWED_USER_IDS is empty, every member of the group can control this server")
	}
	if err := backups.Available(); err != nil {
		log.Printf("warning: backups unavailable: %v", err)
	}

	notify.Send(ctx, i18n.T(i18n.Pick(store.Get().Language, cfg.Language), "main.started",
		html.EscapeString(version), html.EscapeString(cfg.ServerName)))

	tasks := []func(context.Context){
		(&bot.Bot{Cfg: cfg, TG: tg, Store: store, Backup: backups}).Run,
		(&monitor.Monitor{Server: cfg.ServerName, Store: store, Notify: notify, DefaultLang: cfg.Language, DataDir: cfg.DataDir, EnvWhitelist: cfg.LoginWhitelist}).Run,
		backups.Schedule,
		(&services.Watcher{Cfg: cfg, Store: store, Notify: notify}).Run,
		(&httpcheck.Watcher{Cfg: cfg, Store: store, Notify: notify}).Run,
		(&sshwatch.Watcher{Cfg: cfg, Store: store, Notify: notify}).Run,
		(&filewatch.Watcher{Cfg: cfg, Store: store, Notify: notify}).Run,
	}
	if cfg.HeartbeatURL != "" {
		tasks = append(tasks, func(ctx context.Context) { heartbeat(ctx, cfg.HeartbeatURL, cfg.HeartbeatInterval) })
	}

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func() { defer wg.Done(); task(ctx) }()
	}
	log.Printf("vps-alert %s running on %s", version, cfg.ServerName)
	<-ctx.Done()
	log.Printf("shutting down")
	wg.Wait()
	return nil
}

// heartbeat pings an external URL (healthchecks.io, Uptime Kuma push, ...)
// so that a dead server is noticed by someone other than itself.
func heartbeat(ctx context.Context, url string, every time.Duration) {
	client := &http.Client{Timeout: 10 * time.Second}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("heartbeat: %v", err)
			} else {
				resp.Body.Close()
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// pamHook never fails loudly: a broken notifier must not lock anyone out.
func pamHook(envFile string) {
	ev, ok := loginhook.FromEnv()
	if !ok {
		return
	}
	child, err := loginhook.Detach()
	if err != nil || !child {
		return
	}
	if err := config.LoadEnvFile(envFile); err != nil {
		return
	}
	cfg, err := config.Load()
	if err != nil {
		return
	}
	if err := loginhook.Notify(cfg, ev); err != nil {
		log.Printf("pam-hook: %v", err)
	}
}
