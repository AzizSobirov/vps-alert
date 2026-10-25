package i18n

func init() {
	register(RU, map[string]string{
		"bot.not_allowed":        "Нет доступа",
		"bot.unauthorized":       "⚠️ <b>Попытка несанкционированного доступа</b>\nПользователь: %s (<code>%d</code>)\nЧат: <code>%d</code> (%s)\nДействие: <code>%s</code>",
		"bot.unauthorized.added": "добавил бота в чат",
		"bot.unknown":            "Неизвестная команда. См. /help",
		"bot.saved":              "Сохранено",
		"bot.saved_msg":          "✅ Сохранено",
		"bot.off":                "выкл.",
		"bot.unmuted":            "🔔 Оповещения снова включены",

		"bot.btn.back":       "« Назад",
		"bot.btn.backup_now": "▶️ Бэкап сейчас",
		"bot.btn.snapshots":  "📋 Снапшоты",
		"bot.btn.target":     "🎯 Куда",
		"bot.btn.schedule":   "📅 Расписание",
		"bot.btn.databases":  "🗄 Базы данных",
		"bot.btn.rescan":     "🔄 Искать снова",

		"bot.status.muted":          "🔕 Оповещения выключены до %s",
		"bot.status.backup_running": "⏳ Идёт бэкап",

		"bot.backup.title":           "💾 <b>Бэкап</b>",
		"bot.backup.schedule_on":     "📅 Каждые %d дн. в %s\n⏭ Следующий: %s",
		"bot.backup.schedule_off":    "📅 Расписание: выключено",
		"bot.backup.target":          "🎯 Куда: %s",
		"bot.backup.keep":            "🗂 Хранить: %d ежедневных, %d еженедельных, %d ежемесячных",
		"bot.backup.paths_none":      "📁 Папки: <i>нет, добавьте через /paths add /var/www</i>",
		"bot.backup.paths":           "📁 Папки:",
		"bot.backup.dbs_none":        "🗄 Базы данных: <i>нет, выберите через /db</i>",
		"bot.backup.dbs":             "🗄 Базы данных:",
		"bot.backup.last_never":      "🕓 Последний бэкап: ещё не было",
		"bot.backup.last_ok":         "🕓 Последний бэкап: ✅ %s",
		"bot.backup.last_failed":     "🕓 Последний бэкап: ❌ %s",
		"bot.backup.already_running": "Бэкап уже выполняется",
		"bot.backup.started_toast":   "Бэкап запущен",
		"bot.backup.started":         "⏳ Бэкап запущен…",
		"bot.backup.reason_manual":   "вручную, %s",

		"bot.target.title":      "🎯 <b>Куда сохранять бэкапы?</b>",
		"bot.target.local":      "💽 Локально",
		"bot.target.r2":         "☁️ R2",
		"bot.target.both":       "💽+☁️ Оба",
		"bot.target.r2_missing": "R2 не настроен в env-файле",

		"bot.every.title":  "📅 <b>Как часто?</b>\nДругое время: /schedule &lt;дни&gt; &lt;ЧЧ:ММ&gt;",
		"bot.every.daily":  "Каждый день",
		"bot.every.3days":  "Раз в 3 дня",
		"bot.every.weekly": "Раз в неделю",
		"bot.every.off":    "⏸ Выключить",

		"bot.db.scanning":       "🔍 Ищу базы данных…",
		"bot.db.scanning_toast": "Ищу…",
		"bot.db.outdated":       "Список устарел, ищу заново",
		"bot.db.title":          "🗄 <b>Базы данных</b>",
		"bot.db.none":           "Базы данных не найдены. Поддерживаются PostgreSQL и MySQL/MariaDB на сервере или в Docker.",
		"bot.db.hint":           "Нажмите, чтобы включить базу в бэкап или исключить. Каждая выгружается в отдельный файл.",
		"bot.db.not_found":      "(не найдена)",

		"bot.snapshots.title": "📋 <b>Последние снапшоты</b>",
		"bot.snapshots.none":  "<i>снапшотов пока нет</i>",

		"bot.schedule.saved":  "✅ Бэкап каждые %d дн. в %s. Следующий: %s",
		"bot.schedule.off":    "⏸ Автоматические бэкапы выключены",
		"bot.retention.saved": "✅ Хранить %d ежедневных, %d еженедельных и %d ежемесячных снапшотов",
		"bot.report.saved":    "✅ Ежедневный отчёт в %s",
		"bot.report.off":      "⏸ Ежедневный отчёт выключен",
		"bot.mute.saved":      "🔕 Оповещения выключены до %s",

		"bot.lang.choose": "🌐 <b>Выберите язык бота</b>",
		"bot.lang.saved":  "✅ Теперь бот говорит по-русски",

		"bot.state.schedule_on":  "📅 Сейчас: каждые %d дн. в %s, следующий %s",
		"bot.state.schedule_off": "📅 Сейчас: автоматические бэкапы выключены",
		"bot.state.clock":        "🕓 Время сервера: %s",
		"bot.state.retention":    "🗂 Сейчас: %d ежедневных, %d еженедельных, %d ежемесячных",
		"bot.state.report_on":    "📊 Сейчас: каждый день в %s",
		"bot.state.report_off":   "📊 Сейчас: ежедневный отчёт выключен",
		"bot.state.muted":        "🔕 Сейчас: оповещения выключены до %s",
		"bot.state.unmuted":      "🔔 Сейчас: оповещения включены",
		"bot.state.paths":        "📁 Сейчас в бэкапе:",
		"bot.state.paths_none":   "📁 Сейчас: в бэкап ещё ничего не добавлено",
		"bot.state.alerts":       "🚨 <b>Текущие пороги оповещений</b>\nCPU: %s\nRAM: %s\nДиск: %s\nInode: %s\nДлится не меньше: %d мин\nПовторять каждые: %d мин",

		"bot.err.usage":           "Неверный формат, см. примеры ниже.",
		"bot.err.days":            "Число дней должно быть от 1 до 365.",
		"bot.err.clock":           "Время в формате ЧЧ:ММ, например 03:00 (указано: %s).",
		"bot.err.number":          "Неверное число: %s",
		"bot.err.keep_one":        "Нужно хранить хотя бы один снапшот.",
		"bot.err.percent":         "Процент должен быть от 0 до 100.",
		"bot.err.duration":        "Длительность должна быть от 0 до 1440 минут.",
		"bot.err.cooldown":        "Интервал повтора должен быть от 1 до 1440 минут.",
		"bot.err.alert_key":       "Неизвестный параметр %s. Используйте cpu, mem, disk, inode, duration, cooldown или forecast.",
		"bot.err.mute_duration":   "Срок должен быть от 1m до 7d, например 30m, 2h, 1d.",
		"bot.err.path_absolute":   "Путь должен быть абсолютным, например /var/www.",
		"bot.err.path_not_listed": "%s нет в списке.",
		"bot.err.path_listed":     "%s уже в списке.",
		"bot.err.path_missing":    "%s не существует на сервере.",
		"bot.err.lang":            "Неизвестный язык %s. Используйте en, uz или ru.",

		"bot.help.title":          "🛡 <b>vps-alert</b>: мониторинг сервера и бэкапы",
		"bot.help.group.monitor":  "<b>Мониторинг</b>",
		"bot.help.group.backup":   "<b>Бэкапы</b>",
		"bot.help.group.bot":      "<b>Бот</b>",
		"bot.help.group.security": "<b>Безопасность</b>",
		"bot.state.forecast":      "📈 Прогноз диска: предупреждать за %d дн. до заполнения",
		"bot.state.forecast_off":  "📈 Прогноз диска: выключен",
		"bot.err.forecast":        "Дней должно быть от 0 до 90.",
		"bot.help.footer":         "Подробности и примеры по любой команде: <code>/help schedule</code>",
		"bot.help.more":           "📖 <a href=\"%s\">Подробнее в документации</a>",
		"bot.help.unknown":        "Неизвестная команда: %s",

		"help.status.short":    "Состояние сервера",
		"help.alerts.short":    "Пороги оповещений",
		"help.mute.short":      "Временно выключить оповещения",
		"help.unmute.short":    "Включить оповещения",
		"help.report.short":    "Время ежедневного отчёта",
		"help.backup.short":    "Меню бэкапов",
		"help.db.short":        "Базы данных для бэкапа",
		"help.paths.short":     "Папки для бэкапа",
		"help.schedule.short":  "Когда делать бэкап",
		"help.retention.short": "Сколько бэкапов хранить",
		"help.lang.short":      "Язык бота",
		"help.help.short":      "Команды и примеры",

		"help.status": "Показывает CPU, RAM, swap, диски и аптайм прямо сейчас.\n\n" +
			"<b>Пример</b>\n<code>/status</code>: текущее состояние сервера",
		"help.alerts": "Оповещение приходит, когда значение держится выше порога заданное время, и повторяется каждый cooldown, пока остаётся высоким.\n\n" +
			"<b>Использование</b>\n<code>/alerts &lt;параметр&gt; &lt;значение&gt;</code>\n" +
			"Параметры: cpu, mem, disk, inode (в %%), duration, cooldown (в минутах), forecast (за сколько дней до заполнения диска). 0 выключает параметр.\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/alerts cpu 90</code>: оповещать, когда CPU выше 90%%\n" +
			"<code>/alerts disk 80</code>: оповещать, когда диск заполнен на 80%%\n" +
			"<code>/alerts duration 10</code>: только если длится 10 минут\n" +
			"<code>/alerts mem 0</code>: без оповещений о RAM\n" +
			"<code>/alerts forecast 7</code>: предупредить, если диск заполнится в течение 7 дней",
		"help.mute": "Временно выключает оповещения о ресурсах, например на время работ. Оповещения о входе и отчёты о бэкапах продолжают приходить.\n\n" +
			"<b>Использование</b>\n<code>/mute &lt;срок&gt;</code>: m минуты, h часы, d дни, до 7d\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/mute 30m</code>: на 30 минут\n" +
			"<code>/mute 2h</code>: на 2 часа\n" +
			"<code>/mute 1d</code>: на сутки\n\n" +
			"Включить раньше срока: /unmute.",
		"help.unmute": "Снова включает оповещения о ресурсах после /mute.\n\n" +
			"<b>Пример</b>\n<code>/unmute</code>: оповещения снова включены",
		"help.report": "Когда отправлять ежедневный отчёт о состоянии. Время по часовому поясу сервера.\n\n" +
			"<b>Использование</b>\n<code>/report &lt;ЧЧ:ММ&gt;</code>\n<code>/report off</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/report 09:00</code>: каждое утро в 09:00\n" +
			"<code>/report 21:30</code>: каждый вечер в 21:30\n" +
			"<code>/report off</code>: без ежедневного отчёта",
		"help.backup": "Открывает меню бэкапов: текущие настройки и кнопки, чтобы сделать бэкап сейчас, посмотреть снапшоты, выбрать, куда и как часто сохранять.\n\n" +
			"<b>Пример</b>\n<code>/backup</code>: открыть меню\n\n" +
			"Что сохранять, задаётся через /paths и /db, когда через /schedule, сколько хранить через /retention.",
		"help.db": "Находит базы PostgreSQL и MySQL/MariaDB на сервере и в Docker. Нажмите на базу, чтобы включить её в бэкап (✅) или пропустить (⬜). Перед каждым бэкапом каждая база выгружается в отдельный файл.\n\n" +
			"<b>Пример</b>\n<code>/db</code>: найти и выбрать базы",
		"help.paths": "Папки и файлы для бэкапа. Путь должен быть абсолютным и существовать на сервере.\n\n" +
			"<b>Использование</b>\n<code>/paths add &lt;путь&gt;</code>\n<code>/paths rm &lt;путь&gt;</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/paths add /var/www</code>: сохранять ваши сайты\n" +
			"<code>/paths add /etc/nginx</code>: сохранять конфиг nginx\n" +
			"<code>/paths rm /var/www</code>: больше не сохранять",
		"help.schedule": "Когда делать автоматические бэкапы. Время по часовому поясу сервера.\n\n" +
			"<b>Использование</b>\n<code>/schedule &lt;дни&gt; &lt;ЧЧ:ММ&gt;</code>\n<code>/schedule off</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/schedule 1 03:00</code>: каждый день в 03:00\n" +
			"<code>/schedule 3 02:30</code>: раз в 3 дня в 02:30\n" +
			"<code>/schedule 7 04:00</code>: раз в неделю в 04:00\n" +
			"<code>/schedule off</code>: без автоматических бэкапов\n\n" +
			"Если в момент бэкапа сервер был выключен, один бэкап выполнится сразу после включения.",
		"help.retention": "Сколько снапшотов хранить. Старые удаляются после каждого бэкапа.\n\n" +
			"<b>Использование</b>\n<code>/retention &lt;ежедн.&gt; &lt;еженед.&gt; &lt;ежемес.&gt;</code>\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/retention 7 4 6</code>: по одному за каждый день недели, каждую неделю месяца и каждый месяц полугода\n" +
			"<code>/retention 14 8 12</code>: больше истории, до года\n" +
			"<code>/retention 3 0 0</code>: только последние 3 дня\n\n" +
			"0 выключает правило. Благодаря дедупликации лишние снапшоты занимают мало места.",
		"help.lang": "Меняет язык бота. Сообщения, кнопки и меню команд переключаются сразу.\n\n" +
			"<b>Использование</b>\n<code>/lang</code>: выбрать кнопками\n<code>/lang &lt;код&gt;</code>: en, uz или ru\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/lang uz</code>: O'zbekcha\n" +
			"<code>/lang ru</code>: Русский\n" +
			"<code>/lang en</code>: English",
		"help.help": "Показывает все команды или подробно объясняет одну.\n\n" +
			"<b>Примеры</b>\n" +
			"<code>/help</code>: все команды\n" +
			"<code>/help schedule</code>: как работает /schedule",
	})
}
