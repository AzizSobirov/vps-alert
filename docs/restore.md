# Restoring

Restores run **only from a shell on the server**. The bot has no restore command, so a stolen phone
or Telegram account cannot overwrite your data.

```sh
sudo vps-alert restore
```

It asks step by step. Press Enter to accept the value in `[brackets]`. At any numbered choice, `q`
quits without changing anything.

```
Restore from:
   1) local  /var/backups/vps-alert
   2) r2     my-backups/vps-alert
Choose [1]:

Snapshot (newest first):
   1) a1b2c3d4  2026-10-25 03:00    3.2 GiB
   2) e5f6a7b8  2026-10-24 03:00    3.1 GiB
Choose [1]:

What to restore:
   1) Files and folders
   2) A database
Choose [1]:
```

Every restore is reported to the chat: **♻️ Restore on my-server by aziz**.

## Files and folders

You pick everything or one of the backed up paths, and a destination directory (default
`/var/tmp/vps-alert-restore-<time>`).

Files are **never written over the live system**. They keep their full path under the destination,
so `/var/www/shop` comes back as `/var/tmp/vps-alert-restore-.../var/www/shop`. Compare and copy what
you need:

```sh
sudo rsync -a /var/tmp/vps-alert-restore-20261025-140000/var/www/shop/ /var/www/shop/
```

## A database

You pick a database from the snapshot, then either:

- **Import into a database on this server**, or
- **Only save the .sql file**, to inspect it or import it yourself.

When importing:

- If the database still exists, the suggested name is `<name>_restored`, so the copy is created
  **next to** the live one and nothing is lost. Check it, then switch your app over or copy the data.
- To replace the live database, type its own name. You must then type the name again to confirm, and
  all its current data is deleted.
- For Docker databases you can choose a different container, for example when the container name
  changed on a new server.

After importing, any errors the database printed are shown. `role "..." does not exist` means the
database users have to be created first; see [databases](databases.md#known-limitations).

## On a new server

If the old server is gone:

1. Set up the new server and install your apps (and database containers) as before.
2. Install vps-alert with the installer. When asked, use the **same R2 settings and the same
   `RESTIC_PASSWORD`** as the old server: edit `/etc/vps-alert/vps-alert.env` after installation and
   restart.
3. Run `sudo vps-alert restore` and choose **r2**.

Without the old `RESTIC_PASSWORD` the backups cannot be opened. See [backup password](backup-password.md).

## Without vps-alert

The backups are ordinary restic repositories. Any machine with restic and the password can restore
them:

```sh
export RESTIC_PASSWORD='...'
export RESTIC_REPOSITORY=s3:https://<account-id>.r2.cloudflarestorage.com/<bucket>/vps-alert
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=auto
# for a local repository instead: export RESTIC_REPOSITORY=/var/backups/vps-alert

restic snapshots                                         # list snapshots
restic ls latest                                         # files in the newest one
restic restore latest --target /tmp/restore              # everything
restic restore latest --target /tmp/restore --include /var/www/shop          # one folder
restic dump latest /var/lib/vps-alert/dumps/postgres/shop.sql > shop.sql     # one database dump
```
