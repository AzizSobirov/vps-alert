# Alerts and reports

## Resource alerts

vps-alert samples CPU, RAM, disk usage and inode usage every 30 seconds.

| Setting | Default | Change with |
|---|---|---|
| CPU | 90% | `/alerts cpu 90` |
| RAM | 90% | `/alerts mem 90` |
| Disk | 85% | `/alerts disk 85` |
| Inodes | 85% | `/alerts inode 85` |
| Must last | 5 minutes | `/alerts duration 5` |
| Repeat every | 60 minutes | `/alerts cooldown 60` |
| Disk forecast | 7 days | `/alerts forecast 7` |

Send `/alerts` to see the current values. Setting a threshold to `0` disables it.

### How an alert fires and clears

1. A value goes above its threshold. Nothing is sent yet.
2. If it stays above for the full **duration** (5 minutes), you get **🔴 CPU: 95% (threshold 90%)**.
   A short spike that drops back earlier never alerts.
3. While it stays high, a reminder **🔴 CPU still high** is sent once per **cooldown** (60 minutes).
4. When the value falls **5 points below** the threshold, you get **✅ CPU back to normal**. The
   5-point margin stops a value hovering around the line from sending a stream of alerts.

Every mounted disk is checked separately. Temporary and virtual filesystems (tmpfs, Docker overlays,
snap images) are skipped.

**Inodes** are the slots for files on a disk. A disk can run out of them while still having free
space, typically because of millions of small cache or session files. Nothing can then be written, so
it is checked like disk space.

When a CPU or RAM alert fires, it lists the top 3 processes for that resource, so you can see the
cause without logging in:

```
🔴 CPU: 97% (threshold 90%)

Top CPU:
  • php-fpm (PID 2214): 182%
  • mysqld (PID 911): 64%
  • node (PID 3051): 12%
```

## Disk forecast

A threshold alert says the disk *is* full. The forecast warns days earlier, while there is still
time to clean up or resize.

Every 10 minutes vps-alert records how much space each disk uses and keeps a week of history in
`/var/lib/vps-alert/disk-history.json`. It fits a straight line through the last 3 days and, if
usage is clearly growing, estimates when the disk will be full:

```
📈 Disk / is filling up: 81% now, full in ~4 days (+3.2 GiB/day)
```

- The alert is sent when the estimate is within `/alerts forecast` days (default 7), at most once a
  day per disk. `/alerts forecast 0` turns it off.
- It needs at least 6 hours of history, so nothing appears right after installation.
- Flat or shrinking usage, and growth under 1 MiB a day, never alert.
- The daily report lists every disk expected to be full within 30 days.

## Top processes

`/top` shows the 10 processes using the most CPU, measured over one second, and the 10 using the
most memory. CPU% is per core, as in `top`: a busy program on 4 cores can show 400%.

## Muting

```
/mute 30m      /mute 2h      /mute 1d      /unmute
```

Muting silences resource alerts, disk forecasts and stale-backup alerts for up to 7 days, for example
during maintenance. Login alerts, backup reports and the daily report are not muted. `/status` shows
when a mute ends.

## Login alerts

Every SSH login sends:

```
🔐 Login on my-server

👤 User: root
🌐 From: 203.0.113.7
🔧 Service: sshd
🕓 2026-10-25 14:03:11 +05
```

They come from PAM: the installer adds one line to `/etc/pam.d/sshd`. Consequences:

- They arrive even when the vps-alert service is stopped.
- The line is marked `optional` and the hook returns at once, so a broken or slow vps-alert never
  delays or blocks a login.
- Key and password logins are both reported.

### Trusted addresses and users

Logins from whitelisted addresses, networks or users are still reported, but **silently**, without
a sound on your phone, and never marked as a new address. They are not hidden: if someone took over
your Telegram account and whitelisted their own address, their logins would still show up.

- Press **✅ It was me, trust this IP** under a login alert, or
- send `/whitelist add 92.63.204.12`, `/whitelist add 10.0.0.0/24`, or
- use **/menu → 🛡 Security → 📋 Whitelist**.

For a **CI deploy account** such as GitHub Actions, which connects from a different address every
run, trust the user instead: `/whitelist add deploy`. See [security](security.md#trusted-addresses-and-users).

Addresses can also be listed in the env file; this works without a restart:

```sh
LOGIN_IP_WHITELIST=203.0.113.7,198.51.100.0/24
```

To turn login alerts off, remove the `vps-alert` line from `/etc/pam.d/sshd`.

Failed login attempts (brute force) are covered in [security](security.md#failed-ssh-logins).

## Daily report

Every day at 09:00 (server time) you get a summary of the server and the last 24 hours:

```
📊 Daily report
🖥 my-server
🟢 CPU: 8% · 🟢 RAM: 61% · 🟡 Disk /: 78%
💾 Last backup: ✅ 2026-10-26 03:00
⏭ Next backup: 2026-10-27 03:00

🔐 SSH, last 24 hours
  ✅ Logins: 3: aziz ×2, root ×1
  🤝 deploy: 14 times (trusted, silent)
  🚫 Failed attempts: 1 163
  ⚠️ Password login is allowed

⚙️ Services: all 4 running
🌐 Websites: 2/2 up
🛡 Last 24h: no file changes or new ports
```

- **Status and backups**: the same as `/status`, the last and next backup, a stale backup warning
  and disks expected to fill up within 30 days.
- **SSH**: logins by user, trusted logins (such as a CI deploy user) on their own line, failed
  attempts, password logins, and a warning while sshd allows password or root password logins.
- **Services and websites**: what is down right now and how many times they went down during the
  day. Shown only when you watch any.
- **Security**: file changes and new ports during the day.

A quiet day gives a short report; problems are marked ⚠️ or 🔴.

```
/report 08:30     change the time
/report off       turn it off
```

## Service messages

- **🟢 vps-alert started** is sent every time the service starts, which also tells you the server
  rebooted.
- **⚠️ Unauthorized access attempt** means someone outside the allowed chat or users tried to use the
  bot. See [Telegram setup](telegram.md#who-can-do-what).
