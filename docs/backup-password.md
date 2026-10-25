# Backup password (`RESTIC_PASSWORD`)

`RESTIC_PASSWORD` is the key that encrypts every backup. Whether a backup sits on the local disk or in
R2, only someone with this password can open it.

## What it protects

Before anything is written, restic encrypts:

- file contents,
- file and folder names,
- sizes, dates and database dumps.

Someone looking into your R2 bucket, whether a Cloudflare employee or someone who stole your R2 keys,
sees only random bytes. They cannot even tell that a file called `shop.sql` exists.

Technically, the repository has a random master key that encrypts the data (AES-256, with
authentication against tampering). Your password unlocks that master key.

## Where it comes from

The installer generates a random 40-character password, prints it **once** and stores it in
`/etc/vps-alert/vps-alert.env` (readable by root only). To see it again:

```sh
sudo grep RESTIC_PASSWORD /etc/vps-alert/vps-alert.env
```

Copy it into a password manager (Bitwarden, 1Password, KeePass, ...) right after installing.

## If it is lost, the backups are lost

There is no recovery. Not restic, not vps-alert, not Cloudflare can open the backups without it.

The dangerous case: the server dies, and the password only existed in its env file. The backups in R2
are intact but unreadable.

To restore onto a new server you need:

1. `RESTIC_PASSWORD`,
2. the R2 settings (`R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `R2_PREFIX`),
3. restic (or vps-alert).

Keep all of them in your password manager.

## What it does not protect against

| Situation | Protected? |
|---|---|
| R2 keys leak | ✅ The data cannot be read |
| The backup disk or bucket is stolen | ✅ |
| The server itself is hacked (attacker gets root) | ❌ The password and the R2 keys are on the server |

An attacker with root could also **delete** the backups, since the server holds R2 keys with write
access. vps-alert has no protection against this yet. Locking objects in the bucket is not a simple
fix either: it also stops the automatic cleanup from deleting old snapshots. Until this is
addressed, the safest extra copy is one that a different machine pulls from the server, with
credentials the server never sees.

## Rules

- **Use a different password on each server.** The installer does this automatically.
- **Do not just edit `RESTIC_PASSWORD` in the env file once backups exist.** The repository is locked
  with the old password, so backups would fail with `wrong password or no key found`. Change it as
  described below.
- The local and R2 repositories use the same password, but each has its own master key.

## Changing the password

Change it in every repository first, then in the env file:

```sh
sudo -s
set -a; . /etc/vps-alert/vps-alert.env; set +a     # loads the current password

# local repository
RESTIC_REPOSITORY=$BACKUP_LOCAL_REPO restic key passwd

# R2 repository, if used
RESTIC_REPOSITORY=s3:https://$R2_ACCOUNT_ID.r2.cloudflarestorage.com/$R2_BUCKET/$R2_PREFIX \
AWS_ACCESS_KEY_ID=$R2_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$R2_SECRET_ACCESS_KEY AWS_DEFAULT_REGION=auto \
restic key passwd
```

`restic key passwd` asks for the new password twice. Use the same new password for both repositories,
then:

```sh
nano /etc/vps-alert/vps-alert.env      # RESTIC_PASSWORD=<new password>
systemctl restart vps-alert
```

Only the master key's lock changes; the data is not re-encrypted, so this takes seconds. Save the new
password in your password manager and run **Backup now** to confirm it works.

## If the password was never set

If `RESTIC_PASSWORD=` is empty (for example after writing the env file by hand), no backup has ever
been made, and `/backup` shows **⚠️ RESTIC_PASSWORD is not set**. Generate one and restart:

```sh
openssl rand -base64 30
sudo nano /etc/vps-alert/vps-alert.env      # RESTIC_PASSWORD=<that value>
sudo systemctl restart vps-alert
```
