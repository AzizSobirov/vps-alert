# vps-alert

Server monitoring, login alerts and backups, controlled entirely from Telegram.

One static binary, no open ports, no dashboard to host.

- 📊 **Daily report**: CPU, RAM, disk, uptime and last backup, every morning
- 🚨 **Resource alerts**: CPU, RAM, disk and inode thresholds, with a minimum duration, repeat cooldown and "back to normal" messages
- 🔐 **Login alerts**: every SSH session, with user and source IP, via PAM (trusted IPs can be whitelisted)
- 💾 **Backups**: encrypted, deduplicated snapshots with [restic](https://restic.net) to a local repository, Cloudflare R2, or both
- 📅 **Scheduling from the bot**: interval, time, paths and retention, all from chat
- 🛡 **Locked to one chat**: the chat ID (and optionally user IDs) come from the env file only; everyone else is ignored and reported
- 💓 **Dead man's switch**: optional heartbeat to healthchecks.io or Uptime Kuma, so a dead server still gets noticed

📖 **[Documentation](docs/README.md)**: how alerts, backups, schedules, retention, databases and restores work.

🇺🇿 **[Hujjatlar o'zbek tilida](docs/uz/README.md)**

## Installation

This guide assumes an **Ubuntu 22.04+ or Debian 12+** server (Ubuntu 24.04 recommended) and that you
can log in to it with SSH as root or as a user with `sudo`. Other systemd-based Linux distributions
work too; only the package commands differ.

Nothing needs to be opened in the firewall: the server connects out to Telegram, never the other way.

### Step 1. Create the Telegram bot

1. In Telegram open [@BotFather](https://t.me/BotFather) and send `/newbot`.
2. Choose a name and a username (it must end in `bot`, e.g. `my_server_alert_bot`).
3. BotFather replies with a **token** like `7012345678:AAH...`. Keep it secret: anyone with it can
   send messages as your bot.
4. Recommended: send `/setjoingroups` to BotFather, pick your bot and choose **Disable**, so nobody
   else can add it to their groups.

Use **one bot per server**. Several servers can post to the same group, but each needs its own bot.

### Step 2. Find your chat ID

The bot only ever talks to one chat. Decide where alerts should go:

**Private chat (simplest).**
Open your new bot in Telegram and press **Start**. A bot cannot message you until you do this.
Then get your user ID from [@userinfobot](https://t.me/userinfobot): it is a positive number like
`123456789`. Use it both as the chat ID and as the allowed user ID.

**A group** (for example, with your team).
1. Create the group and add your bot to it. If you disabled joining groups in step 1, enable it
   briefly with `/setjoingroups`, add the bot, then disable it again.
2. Send `/start` in the group.
3. Open this address in a browser, with your token:
   `https://api.telegram.org/bot<TOKEN>/getUpdates`
4. Find `"chat":{"id":-100...` in the response. The negative number is the chat ID.
5. Collect the user IDs of the people allowed to control the server (each can get theirs from
   [@userinfobot](https://t.me/userinfobot)). Without them, **every group member** can control the
   server.

Do step 3 before installing: once vps-alert runs, it receives the updates and `getUpdates` shows nothing.

### Step 3. Install

Connect to the server and run the installer:

```sh
ssh root@your-server-ip

curl -fsSL https://github.com/AzizSobirov/vps-alert/releases/latest/download/install.sh | sudo sh
```

It downloads a ready-made binary for your server (x86_64 or ARM), checks its SHA-256 checksum and
installs it. Nothing else is needed on the server: no Go, no Python, no Docker.
(If `curl` is missing on a minimal image: `sudo apt install -y curl`.)

<details>
<summary>Alternative: build from source (for developers)</summary>

Building needs [Go](https://go.dev/dl/) 1.26+, but only on the machine that builds. Either build on
the server:

```sh
git clone https://github.com/AzizSobirov/vps-alert.git
cd vps-alert
make install          # builds and runs deploy/install.sh with the result
```

or build on your own computer and copy the binary (better for servers with little RAM):

```sh
make release
scp dist/vps-alert-linux-amd64 deploy/install.sh root@your-server-ip:/tmp/
ssh root@your-server-ip sudo sh /tmp/install.sh /tmp/vps-alert-linux-amd64
```

Use `vps-alert-linux-arm64` for ARM servers (`uname -m` prints `aarch64`).

</details>

### Step 4. Answer the installer

The installer asks three questions:

| Question | Answer |
|---|---|
| Bot token | The token from step 1 |
| Allowed chat ID | The chat ID from step 2 |
| Allowed user IDs | Your user ID; for a group, a comma separated list like `123456789,987654321` |

Then it:

1. installs [restic](https://restic.net), the backup engine;
2. writes the configuration to `/etc/vps-alert/vps-alert.env`;
3. generates and **prints the backup password**;
4. sends a test message to your chat;
5. asks whether to enable SSH login alerts (answer `Y`);
6. starts the `vps-alert` service, which also starts automatically after a reboot.

> ⚠️ **Save the backup password now**, in a password manager. Every backup is encrypted with it.
> If the server is lost and you don't have this password, the backups **cannot be restored by
> anyone**.

Check that the service is running:

```sh
sudo systemctl status vps-alert     # should say "active (running)"
sudo journalctl -u vps-alert -f     # live logs, Ctrl+C to exit
```

### Step 5. First steps in Telegram

Your chat should already show **🟢 vps-alert started**. Now:

1. `/status`: check that the bot answers.
2. Choose what to back up:
   - `/paths add /var/www`: folders with your sites or app files (repeat for each folder);
   - `/db`: tap the databases to include.
3. `/schedule 1 03:00`: back up every day at 03:00 server time.
4. `/backup` → **▶️ Backup now**: run the first backup and wait for the ✅ message.
5. Log out of the server and back in with SSH: a 🔐 login alert should arrive.

Times are in the server's time zone. Check it with `timedatectl` and change it if needed, for example:

```sh
sudo timedatectl set-timezone Asia/Tashkent
sudo systemctl restart vps-alert
```

### Step 6 (optional). More settings

All settings live in `/etc/vps-alert/vps-alert.env`. Edit it with `sudo nano /etc/vps-alert/vps-alert.env`,
then apply with `sudo systemctl restart vps-alert`. See [`.env.example`](.env.example) for every option.

- **Cloudflare R2**: keep a copy of the backups outside the server (strongly recommended, see below).
- **`LOGIN_IP_WHITELIST`**: your own IPs, so your logins don't trigger alerts.
- **`HEARTBEAT_URL`**: a [healthchecks.io](https://healthchecks.io) check URL, so you are alerted
  even when the server is completely down.

### Managing the service

The installer enables the service, so vps-alert **starts automatically after every reboot** and is
restarted within 5 seconds if it ever crashes.

| Action | Command |
|---|---|
| Status | `sudo systemctl status vps-alert` |
| Stop | `sudo systemctl stop vps-alert` |
| Start | `sudo systemctl start vps-alert` |
| Restart (after editing the env file) | `sudo systemctl restart vps-alert` |
| Turn off autostart on boot | `sudo systemctl disable vps-alert` |
| Turn autostart back on | `sudo systemctl enable vps-alert` |
| Stop now and turn off autostart | `sudo systemctl disable --now vps-alert` |
| Live logs | `sudo journalctl -u vps-alert -f` |
| Last 100 log lines | `sudo journalctl -u vps-alert -n 100` |
| Today's logs | `sudo journalctl -u vps-alert --since today` |

Stopping the service stops monitoring, scheduled backups and bot commands. **SSH login alerts keep
working**, because they are sent by PAM, not by the service. To turn those off, remove the
`vps-alert` line from `/etc/pam.d/sshd`.

To silence alerts for a while without stopping anything, send `/mute 2h` to the bot.

### Updating

Run the installer again:

```sh
curl -fsSL https://github.com/AzizSobirov/vps-alert/releases/latest/download/install.sh | sudo sh
```

It replaces the binary and restarts the service. Your configuration, settings and backups are kept
and no questions are asked. If you built from source: `git pull && make install`.

### Uninstalling

```sh
sudo systemctl disable --now vps-alert
sudo rm /etc/systemd/system/vps-alert.service /usr/local/bin/vps-alert
sudo sed -i '/vps-alert/d' /etc/pam.d/sshd
sudo rm -rf /etc/vps-alert /var/lib/vps-alert
```

Local backups in `/var/backups/vps-alert` are left in place; delete them only if you no longer need
them.

### Troubleshooting

| Problem | Fix |
|---|---|
| Test message fails with `chat not found` | Private chat: press **Start** in the bot first. Group: the bot must be a member, and the ID starts with `-`. |
| Test message fails with `Unauthorized` | The token is wrong; copy it again from BotFather. |
| The bot does not answer commands | Run `sudo journalctl -u vps-alert -f`. `Conflict: terminated by other getUpdates request` means the same token is used on another server: create a separate bot. |
| The bot answers nothing and reports "Unauthorized access attempt" | Your user ID is not in `ALLOWED_USER_IDS`. |
| No SSH login alerts | Check that `/etc/pam.d/sshd` contains the vps-alert line and that `UsePAM yes` is set in `/etc/ssh/sshd_config`. |
| Times in messages are wrong | Set the time zone (step 5). |
| Edited the env file but nothing changed | `sudo systemctl restart vps-alert` |
| Backups to R2 fail on an older distribution | The `restic` from `apt` may be too old. Install the latest from [restic releases](https://github.com/restic/restic/releases) into `/usr/local/bin/restic`. |

## Commands

| Command | |
|---|---|
| `/status` | Current CPU, RAM, disk, uptime |
| `/alerts` | Show thresholds; `/alerts cpu 90`, `/alerts disk 85`, `/alerts duration 5`, `/alerts cooldown 60` (0 disables a threshold) |
| `/mute 1h`, `/unmute` | Silence alerts (`30m`, `2h`, `1d`) |
| `/report 09:00`, `/report off` | Daily report time |
| `/backup` | Menu: run now, list snapshots, choose target, schedule |
| `/db` | Find databases and choose which to back up |
| `/paths add /var/www`, `/paths rm /var/www` | Paths to back up |
| `/schedule 1 03:00`, `/schedule off` | Back up every N days at HH:MM |
| `/retention 7 4 6` | Snapshots to keep: daily, weekly, monthly |

## Configuration

Static configuration and secrets live in `/etc/vps-alert/vps-alert.env` (mode `600`); see
[`.env.example`](.env.example) for every option. Settings changed from the bot are stored in
`/var/lib/vps-alert/settings.json`.

### Cloudflare R2

Create a bucket and an R2 API token with *Object Read & Write*, then set `R2_ACCOUNT_ID`,
`R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` and `R2_BUCKET`, restart (`systemctl restart vps-alert`)
and pick **R2** or **Both** under `/backup` → Target.

### Databases

Send `/db`: the bot finds PostgreSQL and MySQL/MariaDB databases on the host and in Docker
containers, and you tap the ones to back up. Before each backup every selected database is dumped
to its own SQL file, stored by restic and then deleted from disk. A failed dump is reported and
does not stop the others.

Access is passwordless: peer authentication as `postgres`, the MySQL root unix socket, and the
`POSTGRES_USER` / `MYSQL_ROOT_PASSWORD` / `MARIADB_ROOT_PASSWORD` (or `*_FILE`) variables of the
official Docker images.

For anything else, set `BACKUP_PRE_COMMAND` to a command that writes into a path in `/paths`.
It can only be set in the env file, never from the bot.

To restore one database:

```sh
restic restore latest --target /tmp/restore --include /var/lib/vps-alert/dumps/postgres/shop.sql
runuser -u postgres -- psql -d shop -f /tmp/restore/var/lib/vps-alert/dumps/postgres/shop.sql
```

## Restoring

Restores run only from a shell on the server; the bot cannot trigger them.

```sh
sudo vps-alert restore
```

It asks step by step: local or R2, which snapshot, then files or a database.

- **Files** are restored into a new directory (default `/var/tmp/vps-alert-restore-<time>`), never over
  the live system. Copy what you need into place.
- **Databases** can be imported or just saved as a `.sql` file. If the database still exists, the
  default target is `<name>_restored` next to it; replacing the live database requires typing its
  name to confirm.

Every restore is reported to the chat.

### Without vps-alert

The backups are plain restic repositories, so they can also be restored on any machine with restic
and the `RESTIC_PASSWORD`:

```sh
export RESTIC_PASSWORD='...'
export RESTIC_REPOSITORY=s3:https://<account-id>.r2.cloudflarestorage.com/<bucket>/vps-alert
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=auto

restic snapshots
restic restore latest --target /tmp/restore
```

## Security model

- Commands are accepted only from `ALLOWED_CHAT_ID`, and if `ALLOWED_USER_IDS` is set, only from
  those users. Anything else is dropped, the bot leaves foreign groups, and the attempt is
  reported to the allowed chat.
- There is no command that runs arbitrary shell code, and restores can only be started from a shell
  on the server.
- Commands queued while the agent was offline for more than 5 minutes are discarded.
- The bot uses long polling, so no port is opened on the server.
- restic runs with a minimal environment and never sees the bot token.
- The login hook detaches from PAM immediately, so a slow or broken Telegram connection never delays
  or blocks a login.

One bot token can be polled by only one server at a time. Use one bot per server; they can all
post to the same group.

## Development

Requires Go 1.26+ and, for the end-to-end backup tests, restic.

```sh
make test
make release   # dist/vps-alert-linux-amd64, dist/vps-alert-linux-arm64
```

### Releasing

Push a version tag. GitHub Actions runs the tests, builds both binaries and publishes them with
`checksums.txt` and `install.sh` as a GitHub release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

## License

MIT
