package i18n

func init() {
	register(EN, map[string]string{
		"whitelist.added":         "✅ <code>%s</code> is now trusted: its logins arrive silently",
		"whitelist.removed":       "✅ <code>%s</code> removed from the whitelist",
		"whitelist.trusted_by":    "✅ <code>%s</code> added to the whitelist by %s. Its logins now arrive silently.",
		"whitelist.none":          "📋 The whitelist is empty",
		"whitelist.title":         "📋 <b>Whitelist</b>",
		"whitelist.env":           "From the env file (change it on the server):",
		"whitelist.user":          "👤 <code>%s</code> (user, any address)",
		"whitelist.err.exists":    "Already on the whitelist.",
		"whitelist.err.invalid":   "%s is not an IP address, network or user name.",
		"whitelist.err.max":       "The whitelist can hold at most %d entries.",
		"whitelist.err.not_found": "%s is not on the whitelist.",
		"help.whitelist.short":    "Trusted IPs and users",
		"help.whitelist":          "Logins from these addresses, networks or users are still reported, but silently, without a sound on your phone, and they are never marked as a new address. A user entry trusts that user from any address, which suits a CI deploy account such as GitHub Actions. You can also press <b>✅ It was me</b> under a login alert.\n\n<b>Usage</b>\n<code>/whitelist add &lt;IP, network or user&gt;</code>\n<code>/whitelist rm &lt;IP, network or user&gt;</code>\n\n<b>Examples</b>\n<code>/whitelist add 92.63.204.12</code>: your office IP\n<code>/whitelist add 10.0.0.0/24</code>: a whole network\n<code>/whitelist add deploy</code>: the CI deploy user, from any address\n<code>/whitelist rm 92.63.204.12</code>: remove an entry",
		"ssh.ok_none":             "✅ <b>No successful SSH logins in the last 24 hours</b>",
		"ssh.ok_title":            "✅ <b>Successful SSH logins in the last 24 hours</b>: %s",
		"ssh.ok_more":             "… and %d more",
		"ssh.method.key":          "key",
		"ssh.method.password":     "password ⚠️",
		"ssh.cfg_title":           "⚙️ <b>SSH settings</b>",
		"ssh.cfg_password_on":     "Password login: allowed ⚠️",
		"ssh.cfg_password_off":    "Password login: off ✅",
		"ssh.cfg_root_yes":        "Root login: allowed, also with a password ⚠️",
		"ssh.cfg_root_key":        "Root login: with a key only",
		"ssh.cfg_root_no":         "Root login: off ✅",
		"ssh.cfg_unknown":         "⚙️ Could not read the SSH settings (sshd -T)",
		"ssh.cfg_hint":            "How to allow keys only, safely: %s",
		"ssh.password_alert":      "⚠️ <b>Password login</b> on %s\n👤 <b>%s</b> from <code>%s</code> at %s\n\nA password can be guessed; logging in with an SSH key is safer. How to switch: %s",
		"ssh.state_pw_on":         "⚠️ Password login alert: on",
		"ssh.state_pw_off":        "⚠️ Password login alert: off",
		"ssh.alert":               "🚨 <b>SSH brute force</b> on %s\n%s failed logins in the last hour",
		"ssh.top_ips":             "Top IPs:",
		"ssh.top_users":           "Usernames tried:",
		"ssh.summary":             "🔐 <b>Failed SSH logins in the last 24 hours</b>: %s",
		"ssh.unavailable":         "⚠️ Could not read the SSH logs: %s",
		"ssh.state_on":            "🚨 Brute force alert: %d or more failed logins in an hour",
		"ssh.state_off":           "🚨 Brute force alert: off",
		"ssh.err.threshold":       "The threshold must be between 1 and 100000.",

		"files.alert":          "🛡 <b>Important files changed</b> on %s",
		"files.created":        "➕ <code>%s</code> created",
		"files.deleted":        "➖ <code>%s</code> deleted",
		"files.modified":       "✏️ <code>%s</code> modified",
		"files.perms":          "🔐 <code>%s</code> permissions: %s → %s",
		"files.user_added":     "👤 New user: <code>%s</code> (uid %s)",
		"files.user_root":      "🚨 User with root rights (uid 0): <code>%s</code>",
		"files.user_removed":   "👤 User removed: <code>%s</code>",
		"files.group_added":    "👥 <code>%s</code> added to group <b>%s</b>",
		"files.group_removed":  "👥 <code>%s</code> removed from group <b>%s</b>",
		"files.key_added":      "🔑 New SSH key: <code>%s</code> %s",
		"files.key_removed":    "🔑 SSH key removed: <code>%s</code> %s",
		"files.shadow_changed": "🔒 Password or account settings changed for: %s",
		"files.state_on":       "🛡 File watch: on, %d files watched",
		"files.state_off":      "🛡 File watch: off",
		"files.last_change":    "Last change: %s, <code>%s</code>",
		"files.no_change":      "No changes recorded yet.",

		"ports.alert":     "🔌 <b>New listening port</b> on %s",
		"ports.title":     "🔌 <b>Listening ports</b>",
		"ports.none":      "No listening ports found.",
		"ports.new":       "🆕",
		"ports.closed":    "(closed)",
		"ports.err":       "⚠️ Could not list ports: %s",
		"ports.state_on":  "🔌 Port watch: on",
		"ports.state_off": "🔌 Port watch: off",

		"help.ssh.short":   "Failed SSH logins",
		"help.files.short": "Watch important files",
		"help.ports.short": "Listening ports",
		"help.ssh": "Shows successful and failed SSH logins of the last 24 hours and whether password and root logins are allowed. Failed logins are counted from the system logs. When they reach the threshold within an hour, an alert with the top IPs and usernames is sent, at most once an hour. /mute does not silence it.\n\n" +
			"<b>Usage</b>\n<code>/ssh</code>: the last 24 hours\n<code>/ssh threshold &lt;number&gt;</code>\n<code>/ssh on</code> · <code>/ssh off</code>\n<code>/ssh password on</code> · <code>/ssh password off</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/ssh</code>: who tried to log in today\n" +
			"<code>/ssh threshold 50</code>: alert at 50 failures per hour\n" +
			"<code>/ssh off</code>: no brute force alerts\n" +
			"<code>/ssh password off</code>: no alert when someone logs in with a password",
		"help.files": "Alerts when security-relevant files change: users and groups, sudoers, SSH keys and config, cron jobs, systemd services. It shows who was added, which key appeared and which permissions changed. /mute does not silence it.\n\n" +
			"<b>Usage</b>\n<code>/files</code>: status\n<code>/files on</code> · <code>/files off</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/files</code>: what is watched and the last change\n" +
			"<code>/files off</code>: pause during planned maintenance\n" +
			"<code>/files on</code>: watch again from the current state",
		"help.ports": "Lists listening ports with their programs, and alerts when a new port opens and stays open. Ports that close and reopen, e.g. on a service restart, do not alert. /mute does not silence it.\n\n" +
			"<b>Usage</b>\n<code>/ports</code>: current ports\n<code>/ports on</code> · <code>/ports off</code>\n\n" +
			"<b>Examples</b>\n" +
			"<code>/ports</code>: what is listening now\n" +
			"<code>/ports off</code>: no new port alerts",
	})
}
