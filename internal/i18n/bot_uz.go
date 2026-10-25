package i18n

func init() {
	register(UZ, map[string]string{
		"bot.not_allowed":        "Ruxsat yo'q",
		"bot.unauthorized":       "⚠️ <b>Ruxsatsiz kirish urinishi</b>\nFoydalanuvchi: %s (<code>%d</code>)\nChat: <code>%d</code> (%s)\nAmal: <code>%s</code>",
		"bot.unauthorized.added": "botni chatga qo'shdi",
		"bot.unknown":            "Noma'lum buyruq. /help ni ko'ring",
		"bot.saved":              "Saqlandi",
		"bot.saved_msg":          "✅ Saqlandi",
		"bot.off":                "o'chiq",
		"bot.unmuted":            "🔔 Alertlar qayta yoqildi",

		"bot.btn.back":       "« Orqaga",
		"bot.btn.backup_now": "▶️ Hozir backup",
		"bot.btn.snapshots":  "📋 Snapshot'lar",
		"bot.btn.target":     "🎯 Qayerga",
		"bot.btn.schedule":   "📅 Jadval",
		"bot.btn.databases":  "🗄 Bazalar",
		"bot.btn.rescan":     "🔄 Qayta skanerlash",

		"bot.status.muted":          "🔕 Alertlar %s gacha o'chirilgan",
		"bot.status.backup_running": "⏳ Backup olinmoqda",

		"bot.backup.title":           "💾 <b>Backup</b>",
		"bot.backup.schedule_on":     "📅 Har %d kunda soat %s da\n⏭ Keyingisi: %s",
		"bot.backup.schedule_off":    "📅 Jadval: o'chiq",
		"bot.backup.target":          "🎯 Qayerga: %s",
		"bot.backup.keep":            "🗂 Saqlanadi: %d kunlik, %d haftalik, %d oylik",
		"bot.backup.paths_none":      "📁 Papkalar: <i>yo'q, /paths add /var/www bilan qo'shing</i>",
		"bot.backup.paths":           "📁 Papkalar:",
		"bot.backup.dbs_none":        "🗄 Bazalar: <i>yo'q, /db bilan tanlang</i>",
		"bot.backup.dbs":             "🗄 Bazalar:",
		"bot.backup.last_never":      "🕓 Oxirgi backup: hali olinmagan",
		"bot.backup.last_ok":         "🕓 Oxirgi backup: ✅ %s",
		"bot.backup.last_failed":     "🕓 Oxirgi backup: ❌ %s",
		"bot.backup.already_running": "Backup allaqachon olinmoqda",
		"bot.backup.started_toast":   "Backup boshlandi",
		"bot.backup.started":         "⏳ Backup boshlandi…",
		"bot.backup.reason_manual":   "qo'lda, %s",

		"bot.target.title":      "🎯 <b>Backuplar qayerga saqlansin?</b>",
		"bot.target.local":      "💽 Local",
		"bot.target.r2":         "☁️ R2",
		"bot.target.both":       "💽+☁️ Ikkalasiga",
		"bot.target.r2_missing": "R2 env faylda sozlanmagan",

		"bot.every.title":  "📅 <b>Qanchalik tez-tez?</b>\nBoshqa vaqt uchun: /schedule &lt;kunlar&gt; &lt;SS:DD&gt;",
		"bot.every.daily":  "Har kuni",
		"bot.every.3days":  "3 kunda bir",
		"bot.every.weekly": "Haftada bir",
		"bot.every.off":    "⏸ O'chirish",

		"bot.db.scanning":       "🔍 Bazalar qidirilmoqda…",
		"bot.db.scanning_toast": "Qidirilmoqda…",
		"bot.db.outdated":       "Ro'yxat eskirgan, qayta skanerlanmoqda",
		"bot.db.title":          "🗄 <b>Bazalar</b>",
		"bot.db.none":           "Baza topilmadi. Qo'llab-quvvatlanadi: PostgreSQL va MySQL/MariaDB, serverda yoki Docker'da.",
		"bot.db.hint":           "Backupga qo'shish yoki chiqarish uchun bosing. Har bir baza alohida faylga dump qilinadi.",
		"bot.db.not_found":      "(topilmadi)",

		"bot.snapshots.title": "📋 <b>Oxirgi snapshot'lar</b>",
		"bot.snapshots.none":  "<i>hali snapshot yo'q</i>",

		"bot.schedule.saved":  "✅ Backup har %d kunda soat %s da. Keyingisi: %s",
		"bot.schedule.off":    "⏸ Avtomatik backup o'chirildi",
		"bot.retention.saved": "✅ Saqlanadi: %d kunlik, %d haftalik va %d oylik snapshot",
		"bot.report.saved":    "✅ Kunlik hisobot soat %s da",
		"bot.report.off":      "⏸ Kunlik hisobot o'chirildi",
		"bot.mute.saved":      "🔕 Alertlar %s gacha o'chirildi",

		"bot.lang.choose": "🌐 <b>Bot tilini tanlang</b>",
		"bot.lang.saved":  "✅ Bot endi o'zbek tilida",

		"bot.state.schedule_on":  "📅 Hozir: har %d kunda soat %s da, keyingisi %s",
		"bot.state.schedule_off": "📅 Hozir: avtomatik backup o'chiq",
		"bot.state.clock":        "🕓 Server vaqti: %s",
		"bot.state.retention":    "🗂 Hozir: %d kunlik, %d haftalik, %d oylik",
		"bot.state.report_on":    "📊 Hozir: har kuni soat %s da",
		"bot.state.report_off":   "📊 Hozir: kunlik hisobot o'chiq",
		"bot.state.muted":        "🔕 Hozir: alertlar %s gacha o'chirilgan",
		"bot.state.unmuted":      "🔔 Hozir: alertlar yoqilgan",
		"bot.state.paths":        "📁 Hozir backup qilinadi:",
		"bot.state.paths_none":   "📁 Hozir: hali hech narsa backup qilinmaydi",
		"bot.state.alerts":       "🚨 <b>Hozirgi alert chegaralari</b>\nCPU: %s\nRAM: %s\nDisk: %s\nInode: %s\nDavom etishi kerak: %d daqiqa\nQayta eslatish: har %d daqiqada",

		"bot.err.usage":           "Format noto'g'ri, quyidagi misollarga qarang.",
		"bot.err.days":            "Kunlar 1 dan 365 gacha son bo'lishi kerak.",
		"bot.err.clock":           "Vaqt SS:DD ko'rinishida bo'lishi kerak, masalan 03:00 (yozilgan: %s).",
		"bot.err.number":          "Noto'g'ri son: %s",
		"bot.err.keep_one":        "Kamida bitta snapshot saqlanishi kerak.",
		"bot.err.percent":         "Foiz 0 dan 100 gacha bo'lishi kerak.",
		"bot.err.duration":        "Davomiylik 0 dan 1440 daqiqagacha bo'lishi kerak.",
		"bot.err.cooldown":        "Qayta eslatish oralig'i 1 dan 1440 daqiqagacha bo'lishi kerak.",
		"bot.err.alert_key":       "Noma'lum sozlama: %s. cpu, mem, disk, inode, duration yoki cooldown ishlating.",
		"bot.err.mute_duration":   "Muddat 1m dan 7d gacha bo'lishi kerak, masalan 30m, 2h, 1d.",
		"bot.err.path_absolute":   "Yo'l to'liq bo'lishi kerak, masalan /var/www.",
		"bot.err.path_not_listed": "%s ro'yxatda yo'q.",
		"bot.err.path_listed":     "%s allaqachon ro'yxatda.",
		"bot.err.path_missing":    "%s serverda mavjud emas.",
		"bot.err.lang":            "Noma'lum til: %s. en, uz yoki ru ishlating.",

		"bot.help.title":         "🛡 <b>vps-alert</b>: server monitoringi va backup",
		"bot.help.group.monitor": "<b>Monitoring</b>",
		"bot.help.group.backup":  "<b>Backup</b>",
		"bot.help.group.bot":     "<b>Bot</b>",
		"bot.help.footer":        "Istalgan buyruq bo'yicha batafsil izoh va misollar: <code>/help schedule</code>",
		"bot.help.more":          "📖 <a href=\"%s\">Batafsil hujjatda</a>",
		"bot.help.unknown":       "Noma'lum buyruq: %s",

		"help.status.short":    "Server holati",
		"help.alerts.short":    "Alert chegaralari",
		"help.mute.short":      "Alertlarni vaqtincha o'chirish",
		"help.unmute.short":    "Alertlarni qayta yoqish",
		"help.report.short":    "Kunlik hisobot vaqti",
		"help.backup.short":    "Backup menyusi",
		"help.db.short":        "Backup qilinadigan bazalar",
		"help.paths.short":     "Backup qilinadigan papkalar",
		"help.schedule.short":  "Backup qachon olinadi",
		"help.retention.short": "Nechta backup saqlanadi",
		"help.lang.short":      "Bot tili",
		"help.help.short":      "Buyruqlar va misollar",

		"help.status": "CPU, RAM, swap, disklar va uptime'ni hozirgi holatda ko'rsatadi.\n\n" +
			"<b>Misol</b>\n<code>/status</code>: serverning hozirgi holati",
		"help.alerts": "Qiymat belgilangan davomiylik davomida chegaradan yuqori tursa alert keladi, yuqoriligicha qolsa har cooldown'da qayta eslatiladi.\n\n" +
			"<b>Ishlatish</b>\n<code>/alerts &lt;sozlama&gt; &lt;qiymat&gt;</code>\n" +
			"Sozlamalar: cpu, mem, disk, inode (%% da), duration, cooldown (daqiqada). 0 chegarani o'chiradi.\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/alerts cpu 90</code>: CPU 90%% dan oshsa alert\n" +
			"<code>/alerts disk 80</code>: disk 80%% to'lsa alert\n" +
			"<code>/alerts duration 10</code>: faqat 10 daqiqa davom etsa\n" +
			"<code>/alerts mem 0</code>: RAM alertlari o'chiq",
		"help.mute": "Resurs alertlarini vaqtincha o'chiradi, masalan texnik ishlar paytida. Login alertlari va backup hisobotlari kelaveradi.\n\n" +
			"<b>Ishlatish</b>\n<code>/mute &lt;muddat&gt;</code>: m daqiqa, h soat, d kun, 7d gacha\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/mute 30m</code>: 30 daqiqaga\n" +
			"<code>/mute 2h</code>: 2 soatga\n" +
			"<code>/mute 1d</code>: bir kunga\n\n" +
			"Muddatidan oldin yoqish uchun: /unmute.",
		"help.unmute": "/mute dan keyin resurs alertlarini qayta yoqadi.\n\n" +
			"<b>Misol</b>\n<code>/unmute</code>: alertlar yana yoqiladi",
		"help.report": "Kunlik holat hisoboti qachon yuborilishi. Vaqt server vaqt zonasi bo'yicha.\n\n" +
			"<b>Ishlatish</b>\n<code>/report &lt;SS:DD&gt;</code>\n<code>/report off</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/report 09:00</code>: har kuni ertalab 09:00 da\n" +
			"<code>/report 21:30</code>: har kuni kechqurun 21:30 da\n" +
			"<code>/report off</code>: kunlik hisobotsiz",
		"help.backup": "Backup menyusini ochadi: joriy sozlamalar va tugmalar: hozir backup olish, snapshot'lar ro'yxati, backup qayerga va qanchalik tez-tez olinishi.\n\n" +
			"<b>Misol</b>\n<code>/backup</code>: menyuni ochish\n\n" +
			"Nimani backup qilish /paths va /db bilan, qachon /schedule bilan, nechtasini saqlash /retention bilan sozlanadi.",
		"help.db": "Serverda va Docker'da PostgreSQL va MySQL/MariaDB bazalarini topadi. Bazani bossangiz, u backup qilinadi (✅) yoki o'tkazib yuboriladi (⬜). Har bir baza har backupdan oldin alohida faylga dump qilinadi.\n\n" +
			"<b>Misol</b>\n<code>/db</code>: bazalarni topish va tanlash",
		"help.paths": "Backup qilinadigan papka va fayllar. Yo'l to'liq bo'lishi va serverda mavjud bo'lishi kerak.\n\n" +
			"<b>Ishlatish</b>\n<code>/paths add &lt;yo'l&gt;</code>\n<code>/paths rm &lt;yo'l&gt;</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/paths add /var/www</code>: saytlaringizni backup qilish\n" +
			"<code>/paths add /etc/nginx</code>: nginx sozlamalarini backup qilish\n" +
			"<code>/paths rm /var/www</code>: backup qilishni to'xtatish",
		"help.schedule": "Avtomatik backup qachon olinishi. Vaqt server vaqt zonasi bo'yicha.\n\n" +
			"<b>Ishlatish</b>\n<code>/schedule &lt;kunlar&gt; &lt;SS:DD&gt;</code>\n<code>/schedule off</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/schedule 1 03:00</code>: har kuni 03:00 da\n" +
			"<code>/schedule 3 02:30</code>: har 3 kunda 02:30 da\n" +
			"<code>/schedule 7 04:00</code>: haftada bir marta 04:00 da\n" +
			"<code>/schedule off</code>: avtomatik backupsiz\n\n" +
			"Backup vaqtida server o'chiq bo'lsa, u yoqilishi bilan bitta backup olinadi.",
		"help.retention": "Nechta snapshot saqlanishi. Eskilari har backupdan keyin o'chiriladi.\n\n" +
			"<b>Ishlatish</b>\n<code>/retention &lt;kunlik&gt; &lt;haftalik&gt; &lt;oylik&gt;</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/retention 7 4 6</code>: bir hafta har kunidan, bir oy har haftasidan, yarim yil har oyidan bittadan\n" +
			"<code>/retention 14 8 12</code>: ko'proq tarix, bir yilgacha\n" +
			"<code>/retention 3 0 0</code>: faqat oxirgi 3 kun\n\n" +
			"0 qoidani o'chiradi. Deduplikatsiya tufayli qo'shimcha snapshot'lar kam joy oladi.",
		"help.lang": "Bot tilini o'zgartiradi. Xabarlar, tugmalar va buyruqlar menyusi darhol almashadi.\n\n" +
			"<b>Ishlatish</b>\n<code>/lang</code>: tugmalar bilan tanlash\n<code>/lang &lt;kod&gt;</code>: en, uz yoki ru\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/lang uz</code>: O'zbekcha\n" +
			"<code>/lang ru</code>: Русский\n" +
			"<code>/lang en</code>: English",
		"help.help": "Barcha buyruqlar ro'yxatini ko'rsatadi yoki bittasini batafsil tushuntiradi.\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/help</code>: barcha buyruqlar\n" +
			"<code>/help schedule</code>: /schedule qanday ishlaydi",
	})
}
