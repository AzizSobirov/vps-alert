package i18n

// Bot messages. Keys under "bot.err." are plain text (they are escaped when
// shown); "help.<command>.short" is plain text too, since it also becomes
// the Telegram command menu. Everything else is Telegram HTML.
func init() {
	register(EN, map[string]string{
		"bot.not_allowed":        "Not allowed",
		"bot.unauthorized":       "⚠️ <b>Unauthorized access attempt</b>\nUser: %s (<code>%d</code>)\nChat: <code>%d</code> (%s)\nAction: <code>%s</code>",
		"bot.unauthorized.added": "added the bot to a chat",
		"bot.unknown":            "Unknown command. See /help",
		"bot.saved":              "Saved",
		"bot.saved_msg":          "✅ Saved",
		"bot.off":                "off",
		"bot.unmuted":            "🔔 Alerts are back on",

		"bot.btn.back":       "« Back",
		"bot.btn.backup_now": "▶️ Backup now",
		"bot.btn.snapshots":  "📋 Snapshots",
		"bot.btn.target":     "🎯 Target",
		"bot.btn.schedule":   "📅 Schedule",
		"bot.btn.databases":  "🗄 Databases",
		"bot.btn.rescan":     "🔄 Rescan",

		"bot.status.muted":          "🔕 Alerts muted until %s",
		"bot.status.backup_running": "⏳ Backup in progress",

		"bot.backup.title":           "💾 <b>Backup</b>",
		"bot.backup.schedule_on":     "📅 Every %d day(s) at %s\n⏭ Next: %s",
		"bot.backup.schedule_off":    "📅 Schedule: off",
		"bot.backup.target":          "🎯 Target: %s",
		"bot.backup.keep":            "🗂 Keep: %d daily, %d weekly, %d monthly",
		"bot.backup.paths_none":      "📁 Paths: <i>none, add with /paths add /var/www</i>",
		"bot.backup.paths":           "📁 Paths:",
		"bot.backup.dbs_none":        "🗄 Databases: <i>none, choose with /db</i>",
		"bot.backup.dbs":             "🗄 Databases:",
		"bot.backup.last_never":      "🕓 Last run: never",
		"bot.backup.last_ok":         "🕓 Last run: ✅ %s",
		"bot.backup.last_failed":     "🕓 Last run: ❌ %s",
		"bot.backup.already_running": "A backup is already running",
		"bot.backup.started_toast":   "Backup started",
		"bot.backup.started":         "⏳ Backup started…",
		"bot.backup.reason_manual":   "manual, %s",

		"bot.target.title":      "🎯 <b>Where should backups go?</b>",
		"bot.target.local":      "💽 Local",
		"bot.target.r2":         "☁️ R2",
		"bot.target.both":       "💽+☁️ Both",
		"bot.target.r2_missing": "R2 is not configured in the env file",

		"bot.every.title":  "📅 <b>How often?</b>\nFor a custom time use /schedule &lt;days&gt; &lt;HH:MM&gt;",
		"bot.every.daily":  "Daily",
		"bot.every.3days":  "Every 3 days",
		"bot.every.weekly": "Weekly",
		"bot.every.off":    "⏸ Off",

		"bot.db.scanning":       "🔍 Scanning for databases…",
		"bot.db.scanning_toast": "Scanning…",
		"bot.db.outdated":       "The list is outdated, scanning again",
		"bot.db.title":          "🗄 <b>Databases</b>",
		"bot.db.none":           "No databases found. Supported: PostgreSQL and MySQL/MariaDB, on the server or in Docker.",
		"bot.db.hint":           "Tap to include or exclude from backups. Each one is dumped to its own file.",
		"bot.db.not_found":      "(not found)",

		"bot.snapshots.title": "📋 <b>Recent snapshots</b>",
		"bot.snapshots.none":  "<i>no snapshots yet</i>",

		"bot.schedule.saved":  "✅ Backup every %d day(s) at %s. Next: %s",
		"bot.schedule.off":    "⏸ Scheduled backups turned off",
		"bot.retention.saved": "✅ Keeping %d daily, %d weekly and %d monthly snapshots",
		"bot.report.saved":    "✅ Daily report at %s",
		"bot.report.off":      "⏸ Daily report turned off",
		"bot.mute.saved":      "🔕 Alerts muted until %s",

		"bot.lang.choose": "🌐 <b>Choose the bot language</b>",
		"bot.lang.saved":  "✅ The bot now speaks English",

		"bot.state.schedule_on":  "📅 Now: every %d day(s) at %s, next run %s",
		"bot.state.schedule_off": "📅 Now: automatic backups are off",
		"bot.state.clock":        "🕓 Server time: %s",
		"bot.state.retention":    "🗂 Now: %d daily, %d weekly, %d monthly",
		"bot.state.report_on":    "📊 Now: every day at %s",
		"bot.state.report_off":   "📊 Now: the daily report is off",
		"bot.state.muted":        "🔕 Now: alerts are muted until %s",
		"bot.state.unmuted":      "🔔 Now: alerts are on",
		"bot.state.paths":        "📁 Backed up now:",
		"bot.state.paths_none":   "📁 Now: nothing is backed up yet",
		"bot.state.alerts":       "🚨 <b>Alert thresholds now</b>\nCPU: %s\nRAM: %s\nDisk: %s\nInodes: %s\nMust last: %d min\nRepeat every: %d min",

		"bot.err.usage":           "Wrong format, see the examples below.",
		"bot.err.days":            "Days must be a number from 1 to 365.",
		"bot.err.clock":           "Time must be HH:MM, e.g. 03:00 (got %s).",
		"bot.err.number":          "Not a valid number: %s",
		"bot.err.keep_one":        "Keep at least one snapshot.",
		"bot.err.percent":         "Percent must be from 0 to 100.",
		"bot.err.duration":        "Duration must be from 0 to 1440 minutes.",
		"bot.err.cooldown":        "Cooldown must be from 1 to 1440 minutes.",
		"bot.err.alert_key":       "Unknown setting %s. Use cpu, mem, disk, inode, duration or cooldown.",
		"bot.err.mute_duration":   "Duration must be from 1m to 7d, e.g. 30m, 2h, 1d.",
		"bot.err.path_absolute":   "The path must be absolute, e.g. /var/www.",
		"bot.err.path_not_listed": "%s is not in the list.",
		"bot.err.path_listed":     "%s is already in the list.",
		"bot.err.path_missing":    "%s does not exist on the server.",
		"bot.err.lang":            "Unknown language %s. Use en, uz or ru.",

		"bot.help.title":         "🛡 <b>vps-alert</b>: server monitoring and backups",
		"bot.help.group.monitor": "<b>Monitoring</b>",
		"bot.help.group.backup":  "<b>Backups</b>",
		"bot.help.group.bot":     "<b>Bot</b>",
		"bot.help.footer":        "Details and examples for any command: <code>/help schedule</code>",
		"bot.help.more":          "📖 <a href=\"%s\">More in the documentation</a>",
		"bot.help.unknown":       "Unknown command: %s",

		"help.status.short":    "Server status",
		"help.alerts.short":    "Alert thresholds",
		"help.mute.short":      "Silence alerts for a while",
		"help.unmute.short":    "Turn alerts back on",
		"help.report.short":    "Daily report time",
		"help.backup.short":    "Backup menu",
		"help.db.short":        "Databases to back up",
		"help.paths.short":     "Folders to back up",
		"help.schedule.short":  "When backups run",
		"help.retention.short": "How many backups to keep",
		"help.lang.short":      "Bot language",
		"help.help.short":      "Commands and examples",

		"help.status": "Shows CPU, RAM, swap, disks and uptime right now.\n\n" +
			"<b>Example</b>\n<code>/status</code>: current server status",
		"help.alerts": "An alert is sent when a value stays above its threshold for the set duration, and repeated every cooldown while it stays high.\n\n" +
			"<b>Usage</b>\n<code>/alerts &lt;setting&gt; &lt;value&gt;</code>\n" +
			"Settings: cpu, mem, disk, inode (in %%), duration, cooldown (in minutes). 0 turns a threshold off.\n\n" +
			"<b>Examples</b>\n" +
			"<code>/alerts cpu 90</code>: alert when CPU is above 90%%\n" +
			"<code>/alerts disk 80</code>: alert when a disk is 80%% full\n" +
			"<code>/alerts duration 10</code>: only if it lasts 10 minutes\n" +
			"<code>/alerts mem 0</code>: no RAM alerts",
		"help.mute": "Silences resource alerts for a while, e.g. during maintenance. Login alerts and backup reports still arrive.\n\n" +
			"<b>Usage</b>\n<code>/mute &lt;duration&gt;</code>: m minutes, h hours, d days, up to 7d\n\n" +
			"<b>Examples</b>\n" +
			"<code>/mute 30m</code>: for 30 minutes\n" +
			"<code>/mute 2h</code>: for 2 hours\n" +
			"<code>/mute 1d</code>: for a day\n\n" +
			"Turn alerts back on earlier with /unmute.",
		"help.unmute": "Turns resource alerts back on after /mute.\n\n" +
			"<b>Example</b>\n<code>/unmute</code>: alerts on again",
		"help.report": "When the daily status report is sent. Times use the server's time zone.\n\n" +
			"<b>Usage</b>\n<code>/report &lt;HH:MM&gt;</code>\n<code>/report off</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/report 09:00</code>: every morning at 09:00\n" +
			"<code>/report 21:30</code>: every evening at 21:30\n" +
			"<code>/report off</code>: no daily report",
		"help.backup": "Opens the backup menu: current settings, and buttons to back up now, list snapshots, choose where backups go and how often.\n\n" +
			"<b>Example</b>\n<code>/backup</code>: open the menu\n\n" +
			"What to back up is set with /paths and /db, when with /schedule, how many to keep with /retention.",
		"help.db": "Finds PostgreSQL and MySQL/MariaDB databases on the server and in Docker. Tap a database to back it up (✅) or skip it (⬜). Each one is dumped to its own file before every backup.\n\n" +
			"<b>Example</b>\n<code>/db</code>: scan and choose databases",
		"help.paths": "Folders and files to back up. Paths must be absolute and exist on the server.\n\n" +
			"<b>Usage</b>\n<code>/paths add &lt;path&gt;</code>\n<code>/paths rm &lt;path&gt;</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/paths add /var/www</code>: back up your sites\n" +
			"<code>/paths add /etc/nginx</code>: back up the nginx config\n" +
			"<code>/paths rm /var/www</code>: stop backing it up",
		"help.schedule": "When automatic backups run. Times use the server's time zone.\n\n" +
			"<b>Usage</b>\n<code>/schedule &lt;days&gt; &lt;HH:MM&gt;</code>\n<code>/schedule off</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/schedule 1 03:00</code>: every day at 03:00\n" +
			"<code>/schedule 3 02:30</code>: every 3 days at 02:30\n" +
			"<code>/schedule 7 04:00</code>: once a week at 04:00\n" +
			"<code>/schedule off</code>: no automatic backups\n\n" +
			"If the server was off at backup time, one backup runs as soon as it is back.",
		"help.retention": "How many snapshots to keep. Older ones are deleted after each backup.\n\n" +
			"<b>Usage</b>\n<code>/retention &lt;daily&gt; &lt;weekly&gt; &lt;monthly&gt;</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/retention 7 4 6</code>: one per day for a week, one per week for a month, one per month for half a year\n" +
			"<code>/retention 14 8 12</code>: more history, up to a year\n" +
			"<code>/retention 3 0 0</code>: only the last 3 days\n\n" +
			"0 turns a rule off. Thanks to deduplication, extra snapshots take little space.",
		"help.lang": "Changes the bot's language. Messages, buttons and the command menu switch at once.\n\n" +
			"<b>Usage</b>\n<code>/lang</code>: choose with buttons\n<code>/lang &lt;code&gt;</code>: en, uz or ru\n\n" +
			"<b>Examples</b>\n" +
			"<code>/lang uz</code>: O'zbekcha\n" +
			"<code>/lang ru</code>: Русский\n" +
			"<code>/lang en</code>: English",
		"help.help": "Lists all commands, or explains one in detail.\n\n" +
			"<b>Examples</b>\n" +
			"<code>/help</code>: all commands\n" +
			"<code>/help schedule</code>: how /schedule works",
	})
}
