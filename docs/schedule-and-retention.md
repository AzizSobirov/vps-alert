# Schedule and retention

`/schedule` decides **when** backups are taken. `/retention` decides **how many** are kept.

## Schedule

```
/schedule 1 03:00
          │   │
          │   └── time of day: 03:00
          └────── interval in days: every 1 day
```

| Command | Meaning |
|---|---|
| `/schedule 1 03:00` | Every day at 03:00 |
| `/schedule 2 03:00` | Every other day at 03:00 |
| `/schedule 7 04:00` | Once a week at 04:00 |
| `/schedule 30 04:00` | Roughly once a month |
| `/schedule off` | No automatic backups |

The interval can be 1 to 365 days. The buttons under `/backup` → **📅 Schedule** (Daily, Every 3
days, Weekly, Off) change the interval and keep the current time.

> After installation the schedule is **off**. Until you send `/schedule`, backups only run when you
> press **Backup now**.

For servers whose data changes daily (sites, databases, uploads), `/schedule 1` is the right choice:
thanks to deduplication a daily backup only adds what changed.

### How it works

1. When you set a schedule, vps-alert computes the next run, the first matching time after now, and
   saves it to `/var/lib/vps-alert/settings.json`.
2. The service checks the clock every 30 seconds.
3. When the time comes, it first saves the following run, then starts the backup. If the server
   crashes mid-backup, the same run is not started again after reboot.

The next run is shown in the `/backup` menu (**⏭ Next**) and in the daily report.

| Situation | What happens |
|---|---|
| Service restarted | The schedule continues; it is stored on disk |
| Server was off at backup time | One backup runs as soon as it is back, then the schedule continues. A week offline means one backup, not seven |
| A manual backup is still running at the scheduled time | The scheduled run is skipped; the next one runs on time |
| You press **Backup now** | The schedule is not affected |

### Time zone

Times use the **server's** time zone, and many VPS images use UTC. Check and change it:

```sh
timedatectl
sudo timedatectl set-timezone Asia/Tashkent
sudo systemctl restart vps-alert
```

Send `/schedule` again afterwards so the next run is computed in the new zone.

## Retention

```
/retention 7 4 6
           │ │ └── monthly: one per month for the last 6 months
           │ └──── weekly:  one per week for the last 4 weeks
           └────── daily:   one per day for the last 7 days
```

After every backup, snapshots that match none of these rules are deleted, so storage does not grow
forever. Within each day, week or month, the newest snapshot is the one kept.

### Example

Daily backups, `/retention 7 4 6`, today is 25 October:

| Rule | Kept |
|---|---|
| 7 daily | 19, 20, 21, 22, 23, 24, 25 October |
| 4 weekly | the last of each week: 25, 19, 12, 5 October |
| 6 monthly | the last of each month: 25 Oct, 30 Sep, 31 Aug, 31 Jul, 30 Jun, 31 May |

One snapshot can satisfy several rules (25 October is in all three), so this keeps about 14
snapshots, not 17. You can go back to any day of the last week, any week of the last month, and any
month of the last half year.

Older snapshots matter because problems are not always noticed right away. If a table was deleted
three weeks ago, every daily snapshot already contains the damage; a weekly or monthly one does not.

### Storage

Fourteen snapshots are not fourteen full copies. With deduplication, a 3 GB server changing 50 MB a
day needs roughly 4 to 5 GB for all of them.

### Other policies

| Command | Meaning |
|---|---|
| `/retention 7 4 6` | Default: a week of dailies, a month of weeklies, half a year of monthlies |
| `/retention 14 8 12` | More history: two weeks, two months, a year |
| `/retention 3 0 0` | Only the last 3 days, when space is tight |
| `/retention 7 0 3` | A week of dailies and 3 monthlies |

`0` turns a rule off; all three cannot be `0`.

Strictly, "daily 7" means the last 7 *days that have a snapshot*, so when backups run every 3 days,
`7` covers about three weeks.

### Scope

Retention only touches snapshots made by vps-alert on this server: they are tagged `vps-alert` and
carry the server name. Other restic snapshots in the same repository are left alone. If you change
`SERVER_NAME`, snapshots made under the old name are no longer cleaned up automatically.
