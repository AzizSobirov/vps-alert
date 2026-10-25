package i18n

func init() {
	register(RU, map[string]string{
		"ssh.ok_none":          "✅ <b>Успешных входов по SSH за последние 24 часа не было</b>",
		"ssh.ok_title":         "✅ <b>Успешные входы по SSH за последние 24 часа</b>: %s",
		"ssh.ok_more":          "… и ещё %d",
		"ssh.method.key":       "ключ",
		"ssh.method.password":  "пароль ⚠️",
		"ssh.cfg_title":        "⚙️ <b>Настройки SSH</b>",
		"ssh.cfg_password_on":  "Вход по паролю: разрешён ⚠️",
		"ssh.cfg_password_off": "Вход по паролю: выключен ✅",
		"ssh.cfg_root_yes":     "Вход root: разрешён, в том числе по паролю ⚠️",
		"ssh.cfg_root_key":     "Вход root: только по ключу",
		"ssh.cfg_root_no":      "Вход root: выключен ✅",
		"ssh.cfg_unknown":      "⚙️ Не удалось прочитать настройки SSH (sshd -T)",
		"ssh.cfg_hint":         "Как безопасно оставить только вход по ключу: %s",
		"ssh.password_alert":   "⚠️ <b>Вход по паролю</b> на %s\n👤 <b>%s</b> с <code>%s</code> в %s\n\nПароль можно подобрать, вход по SSH-ключу безопаснее. Как перейти: %s",
		"ssh.state_pw_on":      "⚠️ Оповещение о входе по паролю: включено",
		"ssh.state_pw_off":     "⚠️ Оповещение о входе по паролю: выключено",
		"ssh.alert":            "🚨 <b>Перебор паролей SSH</b> на %s\n%s неудачных входов за последний час",
		"ssh.top_ips":          "Самые активные IP:",
		"ssh.top_users":        "Пробовали логины:",
		"ssh.summary":          "🔐 <b>Неудачные входы по SSH за 24 часа</b>: %s",
		"ssh.unavailable":      "⚠️ Не удалось прочитать логи SSH: %s",
		"ssh.state_on":         "🚨 Оповещение о переборе: от %d неудачных входов в час",
		"ssh.state_off":        "🚨 Оповещение о переборе: выключено",
		"ssh.err.threshold":    "Порог должен быть от 1 до 100000.",

		"files.alert":          "🛡 <b>Изменились важные файлы</b> на %s",
		"files.created":        "➕ <code>%s</code> создан",
		"files.deleted":        "➖ <code>%s</code> удалён",
		"files.modified":       "✏️ <code>%s</code> изменён",
		"files.perms":          "🔐 <code>%s</code> права: %s → %s",
		"files.user_added":     "👤 Новый пользователь: <code>%s</code> (uid %s)",
		"files.user_root":      "🚨 Пользователь с правами root (uid 0): <code>%s</code>",
		"files.user_removed":   "👤 Пользователь удалён: <code>%s</code>",
		"files.group_added":    "👥 <code>%s</code> добавлен в группу <b>%s</b>",
		"files.group_removed":  "👥 <code>%s</code> удалён из группы <b>%s</b>",
		"files.key_added":      "🔑 Новый SSH-ключ: <code>%s</code> %s",
		"files.key_removed":    "🔑 SSH-ключ удалён: <code>%s</code> %s",
		"files.shadow_changed": "🔒 Изменены пароль или настройки учётной записи: %s",
		"files.state_on":       "🛡 Наблюдение за файлами: включено, файлов: %d",
		"files.state_off":      "🛡 Наблюдение за файлами: выключено",
		"files.last_change":    "Последнее изменение: %s, <code>%s</code>",
		"files.no_change":      "Изменений пока не было.",

		"ports.alert":     "🔌 <b>Открыт новый порт</b> на %s",
		"ports.title":     "🔌 <b>Открытые порты</b>",
		"ports.none":      "Открытых портов не найдено.",
		"ports.new":       "🆕",
		"ports.closed":    "(закрыт)",
		"ports.err":       "⚠️ Не удалось получить список портов: %s",
		"ports.state_on":  "🔌 Наблюдение за портами: включено",
		"ports.state_off": "🔌 Наблюдение за портами: выключено",

		"help.ssh.short":   "Неудачные входы по SSH",
		"help.files.short": "Наблюдение за важными файлами",
		"help.ports.short": "Открытые порты",
		"help.ssh": "Показывает успешные и неудачные входы по SSH за последние 24 часа и разрешён ли вход по паролю и для root. Неудачные входы считаются по системным логам. Если за час их становится больше порога, приходит оповещение с самыми активными IP и логинами, не чаще раза в час. /mute его не отключает.\n\n" +
			"<b>Использование</b>\n<code>/ssh</code>: последние 24 часа\n<code>/ssh threshold &lt;число&gt;</code>\n<code>/ssh on</code> · <code>/ssh off</code>\n<code>/ssh password on</code> · <code>/ssh password off</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/ssh</code>: кто пытался войти сегодня\n" +
			"<code>/ssh threshold 50</code>: оповещать от 50 попыток в час\n" +
			"<code>/ssh off</code>: без оповещений о переборе\n" +
			"<code>/ssh password off</code>: не оповещать о входе по паролю",
		"help.files": "Оповещает об изменении важных для безопасности файлов: пользователи и группы, sudoers, SSH-ключи и настройки, задания cron, сервисы systemd. Показывает, кто добавлен, какой ключ появился и какие права изменились. /mute его не отключает.\n\n" +
			"<b>Использование</b>\n<code>/files</code>: состояние\n<code>/files on</code> · <code>/files off</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/files</code>: что отслеживается и последнее изменение\n" +
			"<code>/files off</code>: пауза на время плановых работ\n" +
			"<code>/files on</code>: снова следить с текущего состояния",
		"help.ports": "Показывает открытые порты и их программы и оповещает, когда новый порт открылся и остался открытым. Порты, которые закрылись и открылись снова, например при перезапуске сервиса, не вызывают оповещений. /mute его не отключает.\n\n" +
			"<b>Использование</b>\n<code>/ports</code>: текущие порты\n<code>/ports on</code> · <code>/ports off</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/ports</code>: что сейчас слушает\n" +
			"<code>/ports off</code>: без оповещений о новых портах",
	})
}
