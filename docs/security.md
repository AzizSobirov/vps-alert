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

## Successful logins and SSH settings

`/ssh` also lists the **successful** logins of the last 24 hours, with user, address, method and
time, and shows whether sshd allows password and root logins:

```
✅ Successful SSH logins in the last 24 hours: 2
  aziz · 92.63.204.12 · key · 14:02
  root · 10.0.0.5 · password ⚠️ · 03:40

⚙️ SSH settings
Password login: allowed ⚠️
Root login: allowed, also with a password ⚠️
```

The settings come from `sshd -T`, so they include defaults and files in `sshd_config.d/`.

Two alerts go with this, and `/mute` silences neither:

- **First login from a new address.** The usual 🔐 login message gets a 🆕 line when the address
  has never logged in before. On first start vps-alert learns your usual addresses from the last 30
  days of logs, so they are not flagged. The list is kept in `/var/lib/vps-alert/known-ips.json`.
- **Password login.** When someone logs in with a password instead of a key, you get a ⚠️ alert
  within a minute. Turn it off with `/ssh password off` if you use passwords on purpose.

### Allowing keys only

Thousands of bots try passwords every day. With password login off they cannot get in at all, and
the failed login count drops to almost nothing.

1. Make sure you can log in with an SSH key. On your computer: `ssh-copy-id user@your-server-ip`,
   then `ssh user@your-server-ip` must log in without asking for a password.
2. **Keep your current SSH session open**, so you can undo a mistake.
3. On the server:

   ```sh
   printf "PasswordAuthentication no\nPermitRootLogin prohibit-password\n" | sudo tee /etc/ssh/sshd_config.d/10-keys-only.conf
   sudo sshd -t && sudo systemctl restart ssh
   ```

   On RHEL-based systems the service is called `sshd`.
4. In a **new** terminal, log in again. If it works, you are done; `/ssh` now shows both settings ✅.
   If it does not, remove the file from the session that is still open and restart ssh.

Some cloud images set `PasswordAuthentication yes` in a file such as
`/etc/ssh/sshd_config.d/50-cloud-init.conf`. Files are read in name order and the first value wins,
which is why the file above starts with `10-`.

## Trusted addresses and users

`/whitelist` holds addresses, networks and user names whose logins are reported silently and are
never marked as new (see [alerts](alerts.md#trusted-addresses-and-users)).

```
/whitelist add 92.63.204.12     an address
/whitelist add 10.0.0.0/24      a network
/whitelist add deploy           a user, from any address
/whitelist rm 92.63.204.12
```

### CI deploys (GitHub Actions and others)

CI runners connect from a different address on every run, so whitelisting addresses does not work,
and whitelisting all of GitHub's ranges would trust every cloud machine in them. Give the deploys
their own user and trust that user:

1. Create a user just for deploys, e.g. `sudo adduser --disabled-password deploy`, give it only the
   permissions the deploy needs, and add the CI's public key to `/home/deploy/.ssh/authorized_keys`.
2. `/whitelist add deploy`

Deploy logins then arrive silently, are not marked new, and their addresses do not fill the list of
known addresses. A password login by that user still triggers the password alert, so use a key.

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
