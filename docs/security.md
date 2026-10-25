# Security watch

Besides [login alerts](alerts.md#login-alerts), vps-alert watches for three signs of an attack or an
unwanted change: brute force on SSH, changes to important files, and new listening ports.

None of these alerts are silenced by `/mute`: like login alerts, they are about security, not load.

## Failed SSH logins

vps-alert reads sshd's entries from journald (or `/var/log/auth.log` / `/var/log/secure` when
journald has none) and counts failed logins:

- a wrong password for an existing user,
- a login attempt for a user that does not exist,
- a connection dropped during authentication, e.g. a rejected key, when nothing else was logged
  for it.

Each attempt is counted once, even though sshd logs several lines for it.

Every 5 minutes it looks at the last hour. When the failures reach the threshold (100 by default),
you get:

```
🚨 SSH brute force on my-server
1 240 failed logins in the last hour

Top IPs:
  203.0.113.7: 830
  198.51.100.4: 310
Usernames tried:
  root: 900
  admin: 120
```

At most one such alert is sent per hour, also across restarts.

| Command | Meaning |
|---|---|
| `/ssh` | Failed logins in the last 24 hours, with top IPs and usernames |
| `/ssh threshold 50` | Alert at 50 failures per hour |
| `/ssh off` · `/ssh on` | Turn the alert off or back on (100 per hour) |

A server with SSH open to the internet always gets some attempts. They are harmless as long as
password login is off or passwords are strong. The alert is for unusual waves, not the background
noise. If you see the same IPs again and again, consider [fail2ban](https://github.com/fail2ban/fail2ban)
or allowing SSH only from your IPs.

## Important files

These files are checked every minute:

| What | Files |
|---|---|
| Users and groups | `/etc/passwd`, `/etc/group`, `/etc/shadow` |
| Admin rights | `/etc/sudoers`, `/etc/sudoers.d/*` |
| SSH | `/etc/ssh/sshd_config`, `/etc/ssh/sshd_config.d/*`, `/root/.ssh/authorized_keys`, `/home/*/.ssh/authorized_keys` |
| Scheduled jobs | `/etc/crontab`, `/etc/cron.d/*`, `/var/spool/cron/crontabs/*` |
| Services | `/etc/systemd/system/*.service` |
| Library injection | `/etc/ld.so.preload` |

A file being created, deleted, modified, or getting new permissions or a new owner is reported, with
details where they matter:

```
🛡 Important files changed on my-server

✏️ /etc/passwd modified
   👤 New user: mallory (uid 1002)
   🚨 User with root rights (uid 0): toor
✏️ /etc/group modified
   👥 mallory added to group sudo
✏️ /root/.ssh/authorized_keys modified
   🔑 New SSH key: SHA256:j4bhX0W+JGIpAlh1rFmC80zrN/pcjTWFFbLrkfl9kqU ssh-ed25519 laptop
🔐 /etc/sudoers permissions: 0440 0:0 → 0666 0:0
```

- Group changes are shown for the groups that grant admin-like rights: `sudo`, `wheel`, `admin` and
  `docker`.
- SSH keys are identified by the same SHA256 fingerprint `ssh-keygen -lf` prints, so you can match
  them against your own.
- For `/etc/shadow` only the names of the changed accounts are shown. Password hashes are never sent
  or stored; vps-alert keeps a short hash of each line to tell which account changed.
- File contents are never sent.

The first check after installation only records the current state. Changes made while vps-alert
was stopped are reported when it starts again. The state is kept in
`/var/lib/vps-alert/filewatch.json`.

| Command | Meaning |
|---|---|
| `/files` | Status: number of watched files and the last change |
| `/files off` | Pause, e.g. during planned maintenance |
| `/files on` | Watch again, starting from the current state, so changes made while paused are not reported |

## Listening ports

A new program listening on the network is a common sign of a compromise (a backdoor, a miner's
control port) or of a misconfiguration (a database opened to the internet).

vps-alert lists listening TCP ports and bound UDP ports every minute. When a new one appears and is
still there a minute later, you get:

```
🔌 New listening port on my-server

tcp 0.0.0.0:6379 (redis-server)
```

- Ports seen once stay known, so a service restart that closes and reopens its port does not alert.
- Short-lived sockets, and UDP sockets in the ephemeral range (32768–60999) used by DNS and NTP
  clients, are ignored.
- `0.0.0.0` or `[::]` means reachable from every network interface; `127.0.0.1` or `[::1]` only from
  the server itself.

| Command | Meaning |
|---|---|
| `/ports` | Current ports with their programs; 🆕 marks ports not seen before, *(closed)* known ports that are not listening now |
| `/ports off` · `/ports on` | Turn the alert off or back on |
