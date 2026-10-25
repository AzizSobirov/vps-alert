# How backups work

vps-alert does not copy files itself. It drives [restic](https://restic.net), a widely used backup
tool, and adds scheduling, database dumps and Telegram control on top.

## Setting up

```
/paths add /var/www        what to back up (repeat for each folder)
/db                        which databases to dump, see databases.md
/backup → 🎯 Target        where: Local, R2 or Both
/schedule 1 03:00          when, see schedule-and-retention.md
/retention 7 4 6           how many to keep
```

Settings are stored in `/var/lib/vps-alert/settings.json` and survive restarts.

## What happens during a backup

```
1. Checks         restic installed, RESTIC_PASSWORD set, something to back up
2. Pre-command    BACKUP_PRE_COMMAND from the env file, if set
3. Databases      each selected database is dumped to its own .sql file
4. For each target (local and/or R2):
     a. local: at least 1 GiB free on the disk?
     b. first time only: create the repository (restic init)
     c. restic backup: store what changed, encrypted
     d. restic forget --prune: delete snapshots outside the retention policy
5. Clean up       the .sql dumps are deleted from disk
6. Report         result sent to the chat
```

Only one backup runs at a time. If you press **Backup now** while one is running, the bot says so.

## The report

```
✅ Backup finished: my-server (scheduled)

🗄 Databases: 2/2 dumped
✅ postgres · shop: 120.0 MiB
✅ docker:api-db · api: 300.0 MiB

✅ local: snapshot a1b2c3d4, +12.4 MiB new, 3.2 GiB total, 48 new files
✅ r2: snapshot e5f6a7b8, +12.4 MiB new, 3.2 GiB total, 48 new files

⏱ 1m12s
```

- **+12.4 MiB new**: data that was not in any earlier snapshot, i.e. what this backup actually added.
- **3.2 GiB total**: the size of everything backed up.
- A failed database dump or target is shown with ❌ and the error; the rest still runs.
- **⚠️ some files could not be read**: the snapshot was made, but some files were unreadable (usually
  removed while the backup ran).

## Where backups go

| Target | Location |
|---|---|
| Local | `/var/backups/vps-alert` (`BACKUP_LOCAL_REPO`) |
| R2 | `<bucket>/vps-alert/` (`R2_BUCKET`, `R2_PREFIX`) |

A local backup protects against deleted or broken files. It does **not** protect against losing the
server or its disk, because it lives on the same disk. For that, also send backups to R2 and choose
**Both**.

### Setting up Cloudflare R2

1. In the Cloudflare dashboard open **R2** and create a bucket, e.g. `my-backups`.
2. **Manage R2 API Tokens** → **Create API token** with *Object Read & Write*, limited to that bucket.
3. Put the values in `/etc/vps-alert/vps-alert.env`:

   ```sh
   R2_ACCOUNT_ID=...          # shown on the R2 overview page
   R2_ACCESS_KEY_ID=...
   R2_SECRET_ACCESS_KEY=...
   R2_BUCKET=my-backups
   ```

4. `sudo systemctl restart vps-alert`, then `/backup` → **🎯 Target** → **Both**.

Several servers can share one bucket if each has a different `R2_PREFIX`.

## Encryption and deduplication

- Everything is encrypted with `RESTIC_PASSWORD` before it leaves the server: file contents, file
  names, sizes. Nobody with access to the bucket can read it. See [backup password](backup-password.md).
- Files are split into chunks and each chunk is stored once. A 3 GB site that changes by 50 MB a day
  adds about 50 MB per backup, not 3 GB.

## Limitations

- Backups cover the paths and databases you choose, not the whole system. To rebuild a server you
  reinstall the OS and your apps, then restore the data.
- Restores are done from the server's shell, never from the bot. See [restoring](restore.md).
