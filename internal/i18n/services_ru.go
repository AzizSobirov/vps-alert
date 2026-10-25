package i18n

func init() {
	register(RU, map[string]string{
		"services.state.failed":       "упал с ошибкой",
		"services.state.inactive":     "остановлен",
		"services.state.activating":   "запускается",
		"services.state.deactivating": "останавливается",
		"services.state.exited":       "завершился",
		"services.state.dead":         "не работает",
		"services.state.paused":       "на паузе",
		"services.state.restarting":   "перезапускается",
		"services.state.created":      "не запущен",
		"services.state.removing":     "удаляется",
		"services.state.unknown":      "неизвестно",

		"services.down.systemd":    "🔴 Сервис <b>%s</b>: %s",
		"services.down.docker":     "🔴 Контейнер <b>%s</b>: %s (код выхода %d)",
		"services.up.systemd":      "✅ Сервис <b>%s</b> снова работает (простой %s)",
		"services.up.docker":       "✅ Контейнер <b>%s</b> снова работает (простой %s)",
		"services.loop.systemd":    "🔁 Сервис <b>%s</b> постоянно перезапускается: %d раз за 10 минут",
		"services.loop.docker":     "🔁 Контейнер <b>%s</b> постоянно перезапускается: %d раз за 10 минут",
		"services.missing.systemd": "⚠️ Сервис <b>%s</b> не найден. Если он удалён, снимите отметку в /services.",
		"services.missing.docker":  "⚠️ Контейнер <b>%s</b> не найден. Если он удалён, снимите отметку в /services.",

		"services.bot.scanning":       "🔍 Ищу сервисы и контейнеры…",
		"services.bot.scanning_toast": "Поиск…",
		"services.bot.outdated":       "Список устарел, ищу заново",
		"services.bot.title":          "⚙️ <b>Сервисы</b>",
		"services.bot.hint":           "Нажмите, чтобы следить (✅) или не следить (⬜). Если отслеживаемый сервис или контейнер остановится или будет постоянно перезапускаться, придёт оповещение.",
		"services.bot.none":           "Работающие сервисы и Docker-контейнеры не найдены.",
		"services.bot.not_found":      "(не найден)",
		"services.bot.state":          "⚙️ <b>Сейчас отслеживаются</b>",
		"services.bot.state_none":     "⚙️ Пока ничего не отслеживается.",

		"help.services.short": "Оповещение, если сервис или контейнер остановился",
		"help.services": "Следит за сервисами systemd и Docker-контейнерами. Если один из них остановится или будет постоянно перезапускаться, придёт оповещение, а когда снова заработает — сообщение.\n\n" +
			"<b>Пример</b>\n<code>/services</code>: список работающих сервисов и контейнеров, отметьте нужные\n\n" +
			"Короткий перезапуск не считается: оповещение придёт, только если сервис не работает и через 30 секунд.",
	})
}
