package i18n

// Messages sent outside the bot: status snapshot, alerts, daily report,
// backup reports, login alerts, restore reports and service messages.
func init() {
	register(EN, map[string]string{
		"login.new_ip":      "🆕 <b>First login from this IP address</b>. If it was not you, act now.",
		"metrics.os_uptime": "<i>%s</i>, uptime %s",
		"metrics.cpu":       "%s CPU: <b>%.0f%%</b> (%d cores, load %.2f / %.2f / %.2f)",
		"metrics.ram":       "%s RAM: <b>%.0f%%</b> (%s / %s)",
		"metrics.swap":      "   Swap: %s / %s",
		"metrics.disk":      "%s Disk <code>%s</code>: <b>%.0f%%</b> (%s / %s), inodes %.0f%%",

		"monitor.label.cpu":      "CPU",
		"monitor.label.mem":      "RAM",
		"monitor.label.disk":     "Disk %s",
		"monitor.label.inode":    "Inodes %s",
		"monitor.fired":          "🔴 %s: <b>%.0f%%</b> (threshold %.0f%%)",
		"monitor.still_high":     "🔴 %s still high: <b>%.0f%%</b> for %s",
		"monitor.resolved":       "✅ %s back to normal: %.0f%%",
		"monitor.top.cpu":        "Top CPU:",
		"monitor.top.mem":        "Top RAM:",
		"monitor.top.line_cpu":   "  • <code>%s</code> (PID %d): %.0f%%",
		"monitor.top.line_mem":   "  • <code>%s</code> (PID %d): %.0f%% (%s)",
		"monitor.forecast.alert": "📈 %s is filling up: <b>%.0f%%</b> now, full in ~%s (+%s/day)",
		"monitor.forecast.days":  "%d days",
		"monitor.forecast.hours": "%d hours",

		"monitor.report.title":         "📊 <b>Daily report</b>",
		"monitor.report.backup_never":  "💾 Backup: never run",
		"monitor.report.backup_ok":     "💾 Last backup: ✅ %s",
		"monitor.report.backup_failed": "💾 Last backup: ❌ failed %s",
		"monitor.report.next_backup":   "⏭ Next backup: %s",
		"monitor.report.backup_stale":  "⚠️ No successful backup for %s",
		"monitor.report.forecast":      "📈 %s: full in ~%s (+%s/day)",

		"backup.finished":         "✅ <b>Backup finished</b>",
		"backup.failed":           "❌ <b>Backup failed</b>",
		"backup.reason.scheduled": "scheduled",
		"backup.dumps_summary":    "🗄 Databases: %d/%d dumped",
		"backup.target_ok":        "✅ <b>%s</b>: snapshot <code>%s</code>, +%s new, %s total, %d new files",
		"backup.warn.incomplete":  "some files could not be read",
		"backup.warn.forget":      "cleaning old snapshots failed: %s",
		"backup.stale":            "⚠️ <b>No successful backup for %s</b> on %s\nLast successful backup: %s\nCheck /backup and the logs: <code>journalctl -u vps-alert</code>",
		"backup.stale.never":      "never",
		"backup.stale_resolved":   "✅ Backups are working again.",

		"backup.err.running":          "a backup is already running",
		"backup.err.no_restic":        "restic is not installed (apt install restic)",
		"backup.err.no_password":      "RESTIC_PASSWORD is not set in the env file",
		"backup.err.nothing":          "nothing to back up: add paths with /paths add /path or databases with /db",
		"backup.err.pre_command":      "pre-backup command failed: %s",
		"backup.err.manifest":         "could not write the database list: %s",
		"backup.err.all_dumps_failed": "every database dump failed",
		"backup.err.no_r2":            "R2 is not configured (set R2_* in the env file)",
		"backup.err.low_space":        "only %s free on %s",

		"login.alert": "🔐 <b>Login</b> on %s\n\n👤 User: <b>%s</b>\n🌐 From: <code>%s</code>\n🔧 Service: %s\n🕓 %s",
		"login.local": "local",

		"restore.report":        "♻️ <b>Restore</b> on %s by %s\n%s",
		"restore.everything":    "everything",
		"restore.what.files":    "files (%s) from snapshot %s into %s",
		"restore.what.database": "database %s from snapshot %s into %s",
		"restore.what.dump":     "dump %s from snapshot %s to %s",

		"main.started": "🟢 <b>vps-alert %s started</b> on %s",
		"main.test":    "✅ vps-alert test message from <b>%s</b>",
	})
}
