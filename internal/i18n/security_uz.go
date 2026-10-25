package i18n

func init() {
	register(UZ, map[string]string{
		"ssh.alert":         "🚨 <b>SSH brute force</b>: %s\nOxirgi bir soatda %s ta muvaffaqiyatsiz kirish urinishi",
		"ssh.top_ips":       "Eng faol IP'lar:",
		"ssh.top_users":     "Sinab ko'rilgan loginlar:",
		"ssh.summary":       "🔐 <b>Oxirgi 24 soatdagi muvaffaqiyatsiz SSH urinishlari</b>: %s",
		"ssh.unavailable":   "⚠️ SSH loglarini o'qib bo'lmadi: %s",
		"ssh.state_on":      "🚨 Brute force alerti: soatiga %d va undan ko'p urinishda",
		"ssh.state_off":     "🚨 Brute force alerti: o'chiq",
		"ssh.err.threshold": "Chegara 1 dan 100000 gacha bo'lishi kerak.",

		"files.alert":          "🛡 <b>Muhim fayllar o'zgardi</b>: %s",
		"files.created":        "➕ <code>%s</code> yaratildi",
		"files.deleted":        "➖ <code>%s</code> o'chirildi",
		"files.modified":       "✏️ <code>%s</code> o'zgardi",
		"files.perms":          "🔐 <code>%s</code> huquqlari: %s → %s",
		"files.user_added":     "👤 Yangi foydalanuvchi: <code>%s</code> (uid %s)",
		"files.user_root":      "🚨 Root huquqli foydalanuvchi (uid 0): <code>%s</code>",
		"files.user_removed":   "👤 Foydalanuvchi o'chirildi: <code>%s</code>",
		"files.group_added":    "👥 <code>%s</code> <b>%s</b> guruhiga qo'shildi",
		"files.group_removed":  "👥 <code>%s</code> <b>%s</b> guruhidan chiqarildi",
		"files.key_added":      "🔑 Yangi SSH kalit: <code>%s</code> %s",
		"files.key_removed":    "🔑 SSH kalit o'chirildi: <code>%s</code> %s",
		"files.shadow_changed": "🔒 Parol yoki akkaunt sozlamalari o'zgardi: %s",
		"files.state_on":       "🛡 Fayl kuzatuvi: yoqilgan, %d ta fayl kuzatilmoqda",
		"files.state_off":      "🛡 Fayl kuzatuvi: o'chiq",
		"files.last_change":    "Oxirgi o'zgarish: %s, <code>%s</code>",
		"files.no_change":      "Hali o'zgarish qayd etilmagan.",

		"ports.alert":     "🔌 <b>Yangi port ochildi</b>: %s",
		"ports.title":     "🔌 <b>Ochiq portlar</b>",
		"ports.none":      "Ochiq port topilmadi.",
		"ports.new":       "🆕",
		"ports.closed":    "(yopilgan)",
		"ports.err":       "⚠️ Portlar ro'yxatini olib bo'lmadi: %s",
		"ports.state_on":  "🔌 Port kuzatuvi: yoqilgan",
		"ports.state_off": "🔌 Port kuzatuvi: o'chiq",

		"help.ssh.short":   "Muvaffaqiyatsiz SSH urinishlari",
		"help.files.short": "Muhim fayllarni kuzatish",
		"help.ports.short": "Ochiq portlar",
		"help.ssh": "Tizim loglaridan muvaffaqiyatsiz SSH kirishlarini sanaydi. Bir soat ichida ular chegaraga yetsa, eng faol IP'lar va loginlar bilan alert keladi, soatiga ko'pi bilan bir marta. /mute uni o'chirmaydi.\n\n" +
			"<b>Ishlatish</b>\n<code>/ssh</code>: oxirgi 24 soat\n<code>/ssh threshold &lt;son&gt;</code>\n<code>/ssh on</code> · <code>/ssh off</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/ssh</code>: bugun kim kirmoqchi bo'lgan\n" +
			"<code>/ssh threshold 50</code>: soatiga 50 ta urinishda alert\n" +
			"<code>/ssh off</code>: brute force alertlarisiz",
		"help.files": "Xavfsizlik uchun muhim fayllar o'zgarsa alert yuboradi: foydalanuvchilar va guruhlar, sudoers, SSH kalitlari va sozlamalari, cron vazifalari, systemd servislari. Kim qo'shilgani, qaysi kalit paydo bo'lgani va qaysi huquqlar o'zgargani ko'rsatiladi. /mute uni o'chirmaydi.\n\n" +
			"<b>Ishlatish</b>\n<code>/files</code>: holat\n<code>/files on</code> · <code>/files off</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/files</code>: nima kuzatilmoqda va oxirgi o'zgarish\n" +
			"<code>/files off</code>: rejali texnik ishlar paytida to'xtatish\n" +
			"<code>/files on</code>: joriy holatdan qayta kuzatish",
		"help.ports": "Ochiq portlarni dasturlari bilan ko'rsatadi va yangi port ochilib, ochiqligicha qolsa alert yuboradi. Yopilib qayta ochilgan portlar, masalan servis qayta ishga tushganda, alert bermaydi. /mute uni o'chirmaydi.\n\n" +
			"<b>Ishlatish</b>\n<code>/ports</code>: hozirgi portlar\n<code>/ports on</code> · <code>/ports off</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/ports</code>: hozir nima tinglayapti\n" +
			"<code>/ports off</code>: yangi port alertlarisiz",
	})
}
