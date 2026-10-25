package i18n

func init() {
	register(RU, map[string]string{
		"http.down":           "🔴 <b>%s</b> недоступен: %s",
		"http.up":             "✅ <b>%s</b> снова доступен (простой %s)",
		"http.slow":           "🐢 <b>%s</b> отвечает медленно: %s",
		"http.reason.timeout": "нет ответа за 10 секунд",
		"http.reason.dns":     "доменное имя не найдено",
		"http.reason.refused": "соединение отклонено",
		"http.reason.tls":     "проблема с SSL-сертификатом",

		"http.err.usage":     "Используйте /http add <url> или /http rm <url>.",
		"http.err.url":       "%s — неверный адрес. Он должен начинаться с http:// или https://.",
		"http.err.not_found": "%s нет в списке.",
		"http.err.duplicate": "%s уже есть в списке.",
		"http.err.max":       "Можно проверять не больше %d адресов.",

		"http.bot.added":      "✅ %s добавлен. Первая проверка — в течение минуты.",
		"http.bot.removed":    "✅ %s удалён.",
		"http.bot.state":      "🌐 <b>Проверяемые адреса</b>",
		"http.bot.state_none": "🌐 Пока ни один адрес не проверяется.",
		"http.bot.pending":    "(ещё не проверен)",
		"http.bot.down_for":   "уже %s",

		"help.http.short": "Оповещение, если сайт недоступен",
		"help.http": "Открывает каждый адрес раз в минуту. Оповещение приходит, если адрес не отвечает два раза подряд (ошибка, таймаут или HTTP-статус 400 и выше), когда он снова доступен, и если он три раза подряд отвечает дольше 5 секунд.\n\n" +
			"<b>Использование</b>\n<code>/http add &lt;url&gt;</code>\n<code>/http rm &lt;url&gt;</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/http add https://example.com</code>: проверять сайт\n" +
			"<code>/http add https://api.example.com/health</code>: проверять состояние API\n" +
			"<code>/http rm https://example.com</code>: перестать проверять\n" +
			"<code>/http</code>: все адреса и их состояние",
	})
}
