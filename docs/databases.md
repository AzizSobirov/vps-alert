# Databases

Copying a running database's files can capture them half-written. vps-alert instead dumps each
database to a consistent `.sql` file right before the backup.

## Choosing databases

Send `/db` (or press **🗄 Databases** under `/backup`). vps-alert looks for:

| Where | Found by |
|---|---|
| PostgreSQL on the server | `psql` installed and a `postgres` system user |
| MySQL / MariaDB on the server | `mysqld` or `mariadbd` installed |
| Docker containers | running containers whose image is `postgres`, `postgis`, `timescale`, `pgvector`, `mysql`, `mariadb` or `percona` |

It lists every database it can reach. Tap one to switch it between ✅ (backed up) and ⬜ (skipped):

```
✅ postgres · shop
✅ docker:api-db · api
⬜ docker:api-db · test
✅ mysql · wordpress
[🔄 Rescan] [« Back]
```

System databases (`postgres`, `mysql`, `sys`, `information_schema`, `performance_schema`) are not
listed. A selected database that has disappeared is shown as *(not found)* so you can deselect it.

## During a backup

- Each selected database is dumped to its own file under `/var/lib/vps-alert/dumps/`, for example
  `postgres/shop.sql` or `docker-api-db/api.sql`, together with a `manifest.json` describing them.
- restic stores the files with the rest of the backup, then they are deleted from disk.
- A failed dump is reported with ❌ and does not stop the others.

| Engine | Dump command |
|---|---|
| PostgreSQL | `pg_dump` (plain SQL) |
| MySQL / MariaDB | `mysqldump` / `mariadb-dump --single-transaction --routines --triggers --events` |

## Access without passwords

vps-alert does not ask for database passwords. It uses the access root already has:

- **PostgreSQL on the server**: runs as the `postgres` system user (peer authentication).
- **MySQL / MariaDB on the server**: connects as `root` over the local socket (the default on Debian
  and Ubuntu).
- **Docker**: runs the client inside the container with the credentials the official images keep in
  their environment: `POSTGRES_USER`, `MYSQL_ROOT_PASSWORD`, `MARIADB_ROOT_PASSWORD`, or their `_FILE`
  variants.

Database passwords are never entered through the bot, because they would stay in the chat history.

### When a database is not found

Scan warnings are shown under the list. Common causes:

- A MySQL container started with `MYSQL_RANDOM_ROOT_PASSWORD`: there is no password to read.
- MySQL root on the server uses a password instead of socket authentication.
- A remote or managed database (RDS, Supabase, ...): it is not on this server.

For these, write your own dump command into the env file, so that it saves into a folder you added
with `/paths add`:

```sh
BACKUP_PRE_COMMAND=PGPASSWORD=secret pg_dump -h db.example.com -U app app > /var/backups/db/app.sql
```

It runs before every backup and can only be set in the env file, never from the bot.

## Restoring

`sudo vps-alert restore` → **A database** imports a dump back. See [restoring](restore.md#a-database).

## Known limitations

- PostgreSQL roles (users) are not part of `pg_dump`. Restoring onto a new server that lacks the
  roles prints `role "..." does not exist` errors; create the users first, then restore again.
- MongoDB, Redis and SQLite are not detected yet. SQLite files inside a backed up path are copied as
  ordinary files.
