package i18n

func init() {
	register(UZ, map[string]string{
		"whitelist.added":         "✅ <code>%s</code> endi ishonchli: uning kirishlari ovozsiz keladi",
		"whitelist.removed":       "✅ <code>%s</code> oq ro'yxatdan olib tashlandi",
		"whitelist.trusted_by":    "✅ <code>%s</code> ni %s oq ro'yxatga qo'shdi. Uning kirishlari endi ovozsiz keladi.",
		"whitelist.none":          "📋 Oq ro'yxat bo'sh",
		"whitelist.title":         "📋 <b>Oq ro'yxat</b>",
		"whitelist.env":           "Env fayldan (serverda o'zgartiriladi):",
		"whitelist.user":          "👤 <code>%s</code> (foydalanuvchi, istalgan manzildan)",
		"whitelist.err.exists":    "Allaqachon oq ro'yxatda.",
		"whitelist.err.invalid":   "%s IP manzil, tarmoq yoki foydalanuvchi nomi emas.",
		"whitelist.err.max":       "Oq ro'yxatda ko'pi bilan %d ta yozuv bo'lishi mumkin.",
		"whitelist.err.not_found": "%s oq ro'yxatda yo'q.",
		"help.whitelist.short":    "Ishonchli IP va foydalanuvchilar",
		"help.whitelist":          "Bu manzil, tarmoq yoki foydalanuvchilardan kirishlar baribir xabar qilinadi, lekin ovozsiz, telefon jiringlamaydi, va ular hech qachon yangi manzil deb belgilanmaydi. Foydalanuvchi yozuvi uni istalgan manzildan ishonchli qiladi, bu GitHub Actions kabi CI deploy akkaunti uchun qulay. Login xabari ostidagi <b>✅ Bu men</b> tugmasini bosish ham mumkin.\n\n<b>Ishlatish</b>\n<code>/whitelist add &lt;IP, tarmoq yoki foydalanuvchi&gt;</code>\n<code>/whitelist rm &lt;IP, tarmoq yoki foydalanuvchi&gt;</code>\n\n<b>Misollar</b>\n<code>/whitelist add 92.63.204.12</code>: ofisingiz IP'si\n<code>/whitelist add 10.0.0.0/24</code>: butun tarmoq\n<code>/whitelist add deploy</code>: CI deploy foydalanuvchisi, istalgan manzildan\n<code>/whitelist rm 92.63.204.12</code>: yozuvni o'chirish",
		"ssh.ok_none":             "✅ <b>Oxirgi 24 soatda muvaffaqiyatli SSH kirish bo'lmagan</b>",
		"ssh.ok_title":            "✅ <b>Oxirgi 24 soatdagi muvaffaqiyatli SSH kirishlar</b>: %s",
		"ssh.ok_more":             "… va yana %d ta",
		"ssh.method.key":          "kalit",
		"ssh.method.password":     "parol ⚠️",
		"ssh.cfg_title":           "⚙️ <b>SSH sozlamalari</b>",
		"ssh.cfg_password_on":     "Parol bilan kirish: ruxsat etilgan ⚠️",
		"ssh.cfg_password_off":    "Parol bilan kirish: o'chiq ✅",
		"ssh.cfg_root_yes":        "Root kirishi: ruxsat etilgan, parol bilan ham ⚠️",
		"ssh.cfg_root_key":        "Root kirishi: faqat kalit bilan",
		"ssh.cfg_root_no":         "Root kirishi: o'chiq ✅",
		"ssh.cfg_unknown":         "⚙️ SSH sozlamalarini o'qib bo'lmadi (sshd -T)",
		"ssh.cfg_hint":            "Faqat kalit bilan kirishni xavfsiz yoqish: %s",
		"ssh.password_alert":      "⚠️ <b>Parol bilan kirish</b>: %s\n👤 <b>%s</b>, <code>%s</code> dan, soat %s da\n\nParolni topib olish mumkin, SSH kalit bilan kirish xavfsizroq. Qanday o'tish: %s",
		"ssh.state_pw_on":         "⚠️ Parol bilan kirish alerti: yoqiq",
		"ssh.state_pw_off":        "⚠️ Parol bilan kirish alerti: o'chiq",
		"ssh.alert":               "🚨 <b>SSH brute force</b>: %s\nOxirgi bir soatda %s ta muvaffaqiyatsiz kirish urinishi",
		"ssh.top_ips":             "Eng faol IP'lar:",
		"ssh.top_users":           "Sinab ko'rilgan loginlar:",
		"ssh.summary":             "🔐 <b>Oxirgi 24 soatdagi muvaffaqiyatsiz SSH urinishlari</b>: %s",
		"ssh.unavailable":         "⚠️ SSH loglarini o'qib bo'lmadi: %s",
		"ssh.state_on":            "🚨 Brute force alerti: soatiga %d va undan ko'p urinishda",
		"ssh.state_off":           "🚨 Brute force alerti: o'chiq",
		"ssh.err.threshold":       "Chegara 1 dan 100000 gacha bo'lishi kerak.",

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
		"help.ssh": "Oxirgi 24 soatdagi muvaffaqiyatli va muvaffaqiyatsiz SSH kirishlarni hamda parol va root bilan kirishga ruxsat bor-yo'qligini ko'rsatadi. Muvaffaqiyatsiz kirishlar tizim loglaridan sanaladi. Bir soat ichida ular chegaraga yetsa, eng faol IP'lar va loginlar bilan alert keladi, soatiga ko'pi bilan bir marta. /mute uni o'chirmaydi.\n\n" +
			"<b>Ishlatish</b>\n<code>/ssh</code>: oxirgi 24 soat\n<code>/ssh threshold &lt;son&gt;</code>\n<code>/ssh on</code> · <code>/ssh off</code>\n<code>/ssh password on</code> · <code>/ssh password off</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/ssh</code>: bugun kim kirmoqchi bo'lgan\n" +
			"<code>/ssh threshold 50</code>: soatiga 50 ta urinishda alert\n" +
			"<code>/ssh off</code>: brute force alertlarisiz\n" +
			"<code>/ssh password off</code>: parol bilan kirilganda alert yuborilmaydi",
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
