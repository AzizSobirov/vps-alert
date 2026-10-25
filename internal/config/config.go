// Package config loads vps-alert's static configuration from environment
// variables. Secrets and identity live here; anything that can be changed
// from the bot lives in the settings package instead.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken   string
	ChatID     int64
	UserIDs    map[int64]bool // empty means every member of ChatID is trusted
	ServerName string
	DataDir    string
	Language   string // default bot language; /lang overrides it

	ResticBin      string
	ResticPassword string
	LocalRepo      string
	PreBackupCmd   string

	R2AccountID string
	R2AccessKey string
	R2SecretKey string
	R2Bucket    string
	R2Prefix    string

	HeartbeatURL      string
	HeartbeatInterval time.Duration

	LoginWhitelist []*net.IPNet
}

func (c *Config) R2Enabled() bool {
	return c.R2AccountID != "" && c.R2AccessKey != "" && c.R2SecretKey != "" && c.R2Bucket != ""
}

func (c *Config) SettingsPath() string {
	return filepath.Join(c.DataDir, "settings.json")
}

// Load reads the configuration from the process environment.
func Load() (*Config, error) {
	c := &Config{
		BotToken:       os.Getenv("BOT_TOKEN"),
		ServerName:     os.Getenv("SERVER_NAME"),
		DataDir:        envOr("DATA_DIR", "/var/lib/vps-alert"),
		ResticBin:      envOr("RESTIC_BIN", "restic"),
		ResticPassword: os.Getenv("RESTIC_PASSWORD"),
		LocalRepo:      envOr("BACKUP_LOCAL_REPO", "/var/backups/vps-alert"),
		PreBackupCmd:   os.Getenv("BACKUP_PRE_COMMAND"),
		R2AccountID:    os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKey:    os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretKey:    os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:       os.Getenv("R2_BUCKET"),
		R2Prefix:       strings.Trim(envOr("R2_PREFIX", "vps-alert"), "/"),
		HeartbeatURL:   os.Getenv("HEARTBEAT_URL"),
		UserIDs:        map[int64]bool{},
	}

	var errs []error
	if c.BotToken == "" {
		errs = append(errs, errors.New("BOT_TOKEN is required"))
	}

	chat := os.Getenv("ALLOWED_CHAT_ID")
	if chat == "" {
		errs = append(errs, errors.New("ALLOWED_CHAT_ID is required"))
	} else if id, err := strconv.ParseInt(chat, 10, 64); err != nil {
		errs = append(errs, fmt.Errorf("ALLOWED_CHAT_ID: %w", err))
	} else {
		c.ChatID = id
	}

	for _, s := range splitList(os.Getenv("ALLOWED_USER_IDS")) {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("ALLOWED_USER_IDS: %q: %w", s, err))
			continue
		}
		c.UserIDs[id] = true
	}

	c.Language = strings.ToLower(os.Getenv("LANGUAGE"))
	if c.Language != "" && c.Language != "en" && c.Language != "uz" && c.Language != "ru" {
		errs = append(errs, fmt.Errorf("LANGUAGE: must be en, uz or ru, got %q", c.Language))
	}

	if c.ServerName == "" {
		c.ServerName, _ = os.Hostname()
	}

	c.HeartbeatInterval = 5 * time.Minute
	if v := os.Getenv("HEARTBEAT_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < 30*time.Second {
			errs = append(errs, fmt.Errorf("HEARTBEAT_INTERVAL: must be a duration >= 30s, got %q", v))
		} else {
			c.HeartbeatInterval = d
		}
	}

	nets, err := ParseIPList(os.Getenv("LOGIN_IP_WHITELIST"))
	if err != nil {
		errs = append(errs, fmt.Errorf("LOGIN_IP_WHITELIST: %w", err))
	}
	c.LoginWhitelist = nets

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadEnvFile sets variables from a KEY=VALUE file. Variables already present
// in the environment win, so systemd's EnvironmentFile and the file agree.
func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, n)
		}
		key = strings.TrimSpace(key)
		val = unquote(strings.TrimSpace(val))
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return sc.Err()
}

// ParseIPList parses a comma separated list of IPs and CIDRs.
func ParseIPList(s string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, item := range splitList(s) {
		if !strings.Contains(item, "/") {
			ip := net.ParseIP(item)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP %q", item)
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			item = fmt.Sprintf("%s/%d", item, bits)
		}
		_, n, err := net.ParseCIDR(item)
		if err != nil {
			return nil, err
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
