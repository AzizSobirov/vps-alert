package i18n

// The /top command.
func init() {
	register(EN, map[string]string{
		"top.title.cpu": "🔥 <b>Top processes by CPU</b>",
		"top.title.mem": "🧠 <b>Top processes by memory</b>",
		"top.col.user":  "USER",
		"top.col.name":  "COMMAND",
		"top.none":      "<i>nothing to show</i>",

		"help.top.short": "Processes using the most CPU and memory",
		"help.top": "Shows the 10 processes using the most CPU (measured over one second) and the 10 using the most memory. Useful right after a CPU or RAM alert; those alerts already include the top 3.\n\n" +
			"CPU%% is of one core, like in <code>top</code>: a busy program on 4 cores can show 400.\n\n" +
			"<b>Example</b>\n<code>/top</code>: what is loading the server right now",
	})
}
