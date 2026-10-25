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

## Muting

```
/mute 30m      /mute 2h      /mute 1d      /unmute
```

Muting silences resource alerts for up to 7 days, for example during maintenance. Login alerts,
backup reports and the daily report are not muted. `/status` shows when a mute ends.

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

To stop alerts for your own logins, list your IPs or networks in the env file and restart:

```sh
LOGIN_IP_WHITELIST=203.0.113.7,198.51.100.0/24
```

To turn login alerts off, remove the `vps-alert` line from `/etc/pam.d/sshd`.

Failed login attempts are not reported yet.

## Daily report

Every day at 09:00 (server time) you get the current status plus the result of the last backup and
the time of the next one.

```
/report 08:30     change the time
/report off       turn it off
```

## Service messages

- **🟢 vps-alert started** is sent every time the service starts, which also tells you the server
  rebooted.
- **⚠️ Unauthorized access attempt** means someone outside the allowed chat or users tried to use the
  bot. See [Telegram setup](telegram.md#who-can-do-what).
