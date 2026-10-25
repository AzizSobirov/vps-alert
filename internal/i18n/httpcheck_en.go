package i18n

func init() {
	register(EN, map[string]string{
		"http.down":           "🔴 <b>%s</b> is down: %s",
		"http.up":             "✅ <b>%s</b> is up again (down %s)",
		"http.slow":           "🐢 <b>%s</b> is slow: answered in %s",
		"http.reason.timeout": "no answer within 10 seconds",
		"http.reason.dns":     "domain name does not resolve",
		"http.reason.refused": "connection refused",
		"http.reason.tls":     "SSL certificate problem",

		"http.err.usage":     "Use /http add <url> or /http rm <url>.",
		"http.err.url":       "%s is not a valid address. It must start with http:// or https://.",
		"http.err.not_found": "%s is not in the list.",
		"http.err.duplicate": "%s is already in the list.",
		"http.err.max":       "At most %d addresses can be checked.",

		"http.bot.added":      "✅ Added %s. The first check runs within a minute.",
		"http.bot.removed":    "✅ Removed %s.",
		"http.bot.state":      "🌐 <b>Checked addresses</b>",
		"http.bot.state_none": "🌐 No addresses are checked yet.",
		"http.bot.pending":    "(not checked yet)",
		"http.bot.down_for":   "for %s",

		"help.http.short": "Alert when a website is down",
		"help.http": "Opens each address every minute. You get an alert when it fails twice in a row (error, timeout or HTTP status 400 and above), when it is back, and when it answers slower than 5 seconds three times in a row.\n\n" +
			"<b>Usage</b>\n<code>/http add &lt;url&gt;</code>\n<code>/http rm &lt;url&gt;</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/http add https://example.com</code>: check a website\n" +
			"<code>/http add https://api.example.com/health</code>: check an API health endpoint\n" +
			"<code>/http rm https://example.com</code>: stop checking it\n" +
			"<code>/http</code>: show every address and its status",
	})
}
