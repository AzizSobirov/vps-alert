package i18n

func init() {
	register(RU, map[string]string{
		"metrics.os_uptime": "<i>%s</i>, аптайм %s",
		"metrics.cpu":       "%s CPU: <b>%.0f%%</b> (ядер: %d, нагрузка %.2f / %.2f / %.2f)",
		"metrics.ram":       "%s RAM: <b>%.0f%%</b> (%s / %s)",
		"metrics.swap":      "   Swap: %s / %s",
		"metrics.disk":      "%s Диск <code>%s</code>: <b>%.0f%%</b> (%s / %s), иноды %.0f%%",

		"monitor.label.cpu":      "CPU",
		"monitor.label.mem":      "RAM",
		"monitor.label.disk":     "Диск %s",
		"monitor.label.inode":    "Иноды %s",
		"monitor.fired":          "🔴 %s: <b>%.0f%%</b> (порог %.0f%%)",
		"monitor.still_high":     "🔴 %s всё ещё высоко: <b>%.0f%%</b> уже %s",
		"monitor.resolved":       "✅ %s в норме: %.0f%%",
		"monitor.top.cpu":        "Топ по CPU:",
		"monitor.top.mem":        "Топ по RAM:",
		"monitor.top.line_cpu":   "  • <code>%s</code> (PID %d): %.0f%%",
		"monitor.top.line_mem":   "  • <code>%s</code> (PID %d): %.0f%% (%s)",
		"monitor.forecast.alert": "📈 %s заполняется: сейчас <b>%.0f%%</b>, заполнится примерно через %s (+%s в день)",
		"monitor.forecast.days":  "%d дн.",
		"monitor.forecast.hours": "%d ч.",

		"monitor.report.title":         "📊 <b>Ежедневный отчёт</b>",
		"monitor.report.backup_never":  "💾 Бэкап: ещё не выполнялся",
		"monitor.report.backup_ok":     "💾 Последний бэкап: ✅ %s",
		"monitor.report.backup_failed": "💾 Последний бэкап: ❌ ошибка %s",
		"monitor.report.next_backup":   "⏭ Следующий бэкап: %s",
		"monitor.report.backup_stale":  "⚠️ Нет успешного бэкапа уже %s",
		"monitor.report.forecast":      "📈 %s: заполнится примерно через %s (+%s в день)",

		"backup.finished":         "✅ <b>Бэкап завершён</b>",
		"backup.failed":           "❌ <b>Бэкап не удался</b>",
		"backup.reason.scheduled": "по расписанию",
		"backup.dumps_summary":    "🗄 Базы данных: выгружено %d/%d",
		"backup.target_ok":        "✅ <b>%s</b>: снапшот <code>%s</code>, +%s нового, всего %s, новых файлов: %d",
		"backup.warn.incomplete":  "некоторые файлы не удалось прочитать",
		"backup.warn.forget":      "не удалось удалить старые снапшоты: %s",
		"backup.stale":            "⚠️ <b>Нет успешного бэкапа уже %s</b>: %s\nПоследний успешный бэкап: %s\nПроверьте /backup и логи: <code>journalctl -u vps-alert</code>",
		"backup.stale.never":      "никогда",
		"backup.stale_resolved":   "✅ Бэкапы снова работают.",

		"backup.err.running":          "бэкап уже выполняется",
		"backup.err.no_restic":        "restic не установлен (apt install restic)",
		"backup.err.no_password":      "в env-файле не задан RESTIC_PASSWORD",
		"backup.err.nothing":          "нечего сохранять: добавьте папки через /paths add /путь или базы через /db",
		"backup.err.pre_command":      "команда перед бэкапом завершилась с ошибкой: %s",
		"backup.err.manifest":         "не удалось записать список баз: %s",
		"backup.err.all_dumps_failed": "не удалось выгрузить ни одну базу",
		"backup.err.no_r2":            "R2 не настроен (укажите R2_* в env-файле)",
		"backup.err.low_space":        "на %[2]s свободно всего %[1]s",

		"login.alert": "🔐 <b>Вход</b> на %s\n\n👤 Пользователь: <b>%s</b>\n🌐 Откуда: <code>%s</code>\n🔧 Сервис: %s\n🕓 %s",
		"login.local": "локально",

		"restore.report":        "♻️ <b>Восстановление</b> на %s, выполнил %s\n%s",
		"restore.everything":    "всё",
		"restore.what.files":    "файлы (%s) из снапшота %s в %s",
		"restore.what.database": "база %s из снапшота %s в %s",
		"restore.what.dump":     "дамп %s из снапшота %s в %s",

		"main.started": "🟢 <b>vps-alert %s запущен</b> на %s",
		"main.test":    "✅ Тестовое сообщение vps-alert с <b>%s</b>",
	})
}
