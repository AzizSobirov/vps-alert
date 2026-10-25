package i18n

func init() {
	register(UZ, map[string]string{
		"services.state.failed":       "xato bilan to'xtagan",
		"services.state.inactive":     "to'xtatilgan",
		"services.state.activating":   "ishga tushmoqda",
		"services.state.deactivating": "to'xtamoqda",
		"services.state.exited":       "to'xtagan",
		"services.state.dead":         "ishlamayapti",
		"services.state.paused":       "pauza qilingan",
		"services.state.restarting":   "qayta ishga tushmoqda",
		"services.state.created":      "ishga tushirilmagan",
		"services.state.removing":     "o'chirilmoqda",
		"services.state.unknown":      "noma'lum",

		"services.down.systemd":    "🔴 <b>%s</b> servisi %s",
		"services.down.docker":     "🔴 <b>%s</b> konteyneri %s (chiqish kodi %d)",
		"services.up.systemd":      "✅ <b>%s</b> servisi yana ishlayapti (%s to'xtab turdi)",
		"services.up.docker":       "✅ <b>%s</b> konteyneri yana ishlayapti (%s to'xtab turdi)",
		"services.loop.systemd":    "🔁 <b>%s</b> servisi qayta-qayta ishga tushyapti: 10 daqiqada %d marta",
		"services.loop.docker":     "🔁 <b>%s</b> konteyneri qayta-qayta ishga tushyapti: 10 daqiqada %d marta",
		"services.missing.systemd": "⚠️ <b>%s</b> servisi topilmadi. Agar u o'chirilgan bo'lsa, /services da belgisini olib tashlang.",
		"services.missing.docker":  "⚠️ <b>%s</b> konteyneri topilmadi. Agar u o'chirilgan bo'lsa, /services da belgisini olib tashlang.",

		"services.bot.scanning":       "🔍 Servis va konteynerlar qidirilmoqda…",
		"services.bot.scanning_toast": "Qidirilmoqda…",
		"services.bot.outdated":       "Ro'yxat eskirgan, qayta qidirilmoqda",
		"services.bot.title":          "⚙️ <b>Servislar</b>",
		"services.bot.hint":           "Kuzatish (✅) yoki e'tiborsiz qoldirish (⬜) uchun bosing. Kuzatilayotgan servis yoki konteyner to'xtasa yoki qayta-qayta ishga tushsa, alert keladi.",
		"services.bot.none":           "Ishlab turgan servis yoki Docker konteyner topilmadi.",
		"services.bot.not_found":      "(topilmadi)",
		"services.bot.state":          "⚙️ <b>Hozir kuzatilayotganlar</b>",
		"services.bot.state_none":     "⚙️ Hozircha hech narsa kuzatilmayapti.",

		"help.services.short": "Servis yoki konteyner to'xtasa alert",
		"help.services": "systemd servislari va Docker konteynerlarini kuzatadi. Ulardan biri to'xtasa yoki qayta-qayta ishga tushsa alert keladi, yana ishga tushganda xabar keladi.\n\n" +
			"<b>Misol</b>\n<code>/services</code>: ishlab turgan servis va konteynerlar ro'yxati, kuzatiladiganlarini bosing\n\n" +
			"Qisqa restart e'tiborga olinmaydi: alert faqat servis 30 soniyadan keyin ham to'xtagan bo'lsa keladi.",
	})
}
