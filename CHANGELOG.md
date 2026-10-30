# Changelog

All notable changes to vps-alert are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

To upgrade, run the installer again; settings and backups are kept:

```sh
curl -fsSL https://github.com/AzizSobirov/vps-alert/releases/latest/download/install.sh | sudo sh
```

## [Unreleased]

## [0.7.0] - 2026-10-30

### Added
- `ALERT_CHAT_ID` sends alerts to another chat, such as a Telegram channel, while commands stay in
  `ALLOWED_CHAT_ID`. The **✅ It was me** button works there for `ALLOWED_USER_IDS` only.
  `vps-alert test` sends a test message to both chats.

## [0.6.0] - 2026-10-26

### Added
- The daily report now summarizes the last 24 hours: SSH logins by user, trusted logins (such as a
  CI deploy user) on their own line, failed attempts, password logins, a warning while password or
  root password logins are allowed, services and websites that are down and how often they went
  down, and file changes and new ports.

## [0.5.0] - 2026-10-26

### Added
- `/whitelist`: trusted addresses, networks and user names, also in the menu under Security. A user
  entry trusts that user from any address, for CI deploy accounts such as GitHub Actions.
- A **✅ It was me** button under login alerts adds the address to the whitelist.

### Changed
- Logins from whitelisted addresses, including `LOGIN_IP_WHITELIST`, are now reported silently
  instead of not at all, so whitelisting cannot hide a login.

### Upgrade notes
- If you set `LOGIN_IP_WHITELIST`, logins from those addresses now arrive as silent messages.

## [0.4.0] - 2026-10-26

### Added
- `/ssh` lists the successful logins of the last 24 hours with user, address, method (key or
  password) and time.
- `/ssh` shows whether sshd allows password and root logins, read from `sshd -T`, with a link to
  a safe guide for allowing keys only.
- The login alert is marked 🆕 when the address has never logged in before. Known addresses are
  learned from the last 30 days of logs on first start.
- An alert within a minute when someone logs in with a password instead of a key. Turn it off with
  `/ssh password off` or in the menu.

### Upgrade notes
- The password login alert is on by default. If you log in with a password on purpose, send
  `/ssh password off`.

## [0.3.0] - 2026-10-26

### Added
- `/menu` (and `/start`): one message with buttons for every section, changing in place. Values
  such as a folder or a website address are asked for and answered by reply.
- `/services`: alerts when chosen systemd services or Docker containers stop, come back, or keep
  restarting.
- `/http`: website checks every minute, with down, recovered and slow alerts.
- `/ssh`: an alert when failed SSH logins reach a threshold within an hour (default 100), with the
  top addresses and usernames, and a 24-hour summary.
- `/files`: alerts on changes to users, sudo and docker groups, SSH keys, sshd config, cron jobs and
  systemd units.
- `/ports`: alerts on newly opened listening ports.
- Disk-full forecast (`/alerts forecast 7`) and `/top`; CPU and RAM alerts now list the top
  processes.
- An alert when no backup has succeeded within the schedule interval plus 12 hours.
- Documentation in Russian.

### Upgrade notes
- Brute-force, file change and new port alerts are on by default. Security alerts are not silenced
  by `/mute`.
- Services and websites are only watched after you choose them with `/services` and `/http`.

## [0.2.0] - 2026-10-26

### Added
- The bot speaks English, Uzbek and Russian. Choose with `/lang`; the installer asks for the
  default and `LANGUAGE` sets it in the env file.
- `/help` lists every command with an example; `/help <command>` explains one with examples and a
  link to the docs. Commands sent without the arguments they need show the current setting and
  examples.
- Documentation in Uzbek.

### Fixed
- A user name containing `<` or `&` no longer stops the backup report from being sent.

### Upgrade notes
- Existing installs keep English until you send `/lang uz` or `/lang ru`.

## [0.1.0] - 2026-10-25

First release.

### Added
- Daily report and alerts for CPU, RAM, disk and inode usage, with duration, cooldown and
  "back to normal" messages.
- SSH login alerts through PAM, with an IP whitelist.
- Encrypted, deduplicated backups with restic to a local repository, Cloudflare R2 or both,
  scheduled and configured from the bot.
- PostgreSQL and MySQL/MariaDB dumps, on the host or in Docker, chosen with `/db`.
- `vps-alert restore` for files and databases, from the server's shell only.
- The bot answers only the chat and users set in the env file.
- Optional heartbeat for healthchecks.io or Uptime Kuma.
- An installer that verifies the downloaded binary's checksum.

[Unreleased]: https://github.com/AzizSobirov/vps-alert/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/AzizSobirov/vps-alert/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/AzizSobirov/vps-alert/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/AzizSobirov/vps-alert/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/AzizSobirov/vps-alert/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/AzizSobirov/vps-alert/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/AzizSobirov/vps-alert/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/AzizSobirov/vps-alert/releases/tag/v0.1.0
