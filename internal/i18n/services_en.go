package i18n

func init() {
	register(EN, map[string]string{
		"services.state.failed":       "failed",
		"services.state.inactive":     "stopped",
		"services.state.activating":   "starting",
		"services.state.deactivating": "stopping",
		"services.state.exited":       "exited",
		"services.state.dead":         "dead",
		"services.state.paused":       "paused",
		"services.state.restarting":   "restarting",
		"services.state.created":      "not started",
		"services.state.removing":     "being removed",
		"services.state.unknown":      "unknown",

		"services.down.systemd":    "🔴 Service <b>%s</b> is %s",
		"services.down.docker":     "🔴 Container <b>%s</b> is %s (exit code %d)",
		"services.up.systemd":      "✅ Service <b>%s</b> is running again (down %s)",
		"services.up.docker":       "✅ Container <b>%s</b> is running again (down %s)",
		"services.loop.systemd":    "🔁 Service <b>%s</b> keeps restarting: %d restarts in 10 minutes",
		"services.loop.docker":     "🔁 Container <b>%s</b> keeps restarting: %d restarts in 10 minutes",
		"services.missing.systemd": "⚠️ Service <b>%s</b> was not found. If it was removed, deselect it in /services.",
		"services.missing.docker":  "⚠️ Container <b>%s</b> was not found. If it was removed, deselect it in /services.",

		"services.bot.scanning":       "🔍 Looking for services and containers…",
		"services.bot.scanning_toast": "Scanning…",
		"services.bot.outdated":       "The list is outdated, scanning again",
		"services.bot.title":          "⚙️ <b>Services</b>",
		"services.bot.hint":           "Tap to watch (✅) or ignore (⬜). You get an alert when a watched service or container stops or keeps restarting.",
		"services.bot.none":           "No running services or Docker containers found.",
		"services.bot.not_found":      "(not found)",
		"services.bot.state":          "⚙️ <b>Watched now</b>",
		"services.bot.state_none":     "⚙️ Nothing is watched yet.",

		"help.services.short": "Alert when a service or container stops",
		"help.services": "Watches systemd services and Docker containers. You get an alert when one stops or keeps restarting, and a message when it is running again.\n\n" +
			"<b>Example</b>\n<code>/services</code>: list running services and containers, tap the ones to watch\n\n" +
			"A short restart is ignored: an alert is sent only if the service is still down 30 seconds later.",
	})
}
