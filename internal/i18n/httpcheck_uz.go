package i18n

func init() {
	register(UZ, map[string]string{
		"http.down":           "🔴 <b>%s</b> ishlamayapti: %s",
		"http.up":             "✅ <b>%s</b> yana ishlayapti (%s ishlamay turdi)",
		"http.slow":           "🐢 <b>%s</b> sekin ishlayapti: %s da javob berdi",
		"http.reason.timeout": "10 soniya ichida javob bermadi",
		"http.reason.dns":     "domen nomi topilmadi",
		"http.reason.refused": "ulanish rad etildi",
		"http.reason.tls":     "SSL sertifikatida muammo",

		"http.err.usage":     "/http add <url> yoki /http rm <url> ishlating.",
		"http.err.url":       "%s to'g'ri manzil emas. U http:// yoki https:// bilan boshlanishi kerak.",
		"http.err.not_found": "%s ro'yxatda yo'q.",
		"http.err.duplicate": "%s ro'yxatda allaqachon bor.",
		"http.err.max":       "Ko'pi bilan %d ta manzilni tekshirish mumkin.",

		"http.bot.added":      "✅ %s qo'shildi. Birinchi tekshiruv bir daqiqa ichida bo'ladi.",
		"http.bot.removed":    "✅ %s olib tashlandi.",
		"http.bot.state":      "🌐 <b>Tekshirilayotgan manzillar</b>",
		"http.bot.state_none": "🌐 Hozircha hech qanday manzil tekshirilmayapti.",
		"http.bot.pending":    "(hali tekshirilmagan)",
		"http.bot.down_for":   "%s dan beri",

		"help.http.short": "Sayt ishlamay qolsa alert",
		"help.http": "Har bir manzilni har daqiqada ochib ko'radi. Manzil ketma-ket ikki marta ishlamasa (xato, javob bermaslik yoki 400 va undan yuqori HTTP status), yana ishlay boshlaganda va ketma-ket uch marta 5 soniyadan sekin javob berganda alert keladi.\n\n" +
			"<b>Ishlatish</b>\n<code>/http add &lt;url&gt;</code>\n<code>/http rm &lt;url&gt;</code>\n\n" +
			"<b>Misollar</b>\n" +
			"<code>/http add https://example.com</code>: saytni tekshirish\n" +
			"<code>/http add https://api.example.com/health</code>: API holatini tekshirish\n" +
			"<code>/http rm https://example.com</code>: tekshirishni to'xtatish\n" +
			"<code>/http</code>: barcha manzillar va ularning holati",
	})
}
