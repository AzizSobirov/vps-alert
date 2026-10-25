package i18n

func init() {
	register(UZ, map[string]string{
		"top.title.cpu": "🔥 <b>Eng ko'p CPU ishlatayotgan jarayonlar</b>",
		"top.title.mem": "🧠 <b>Eng ko'p xotira ishlatayotgan jarayonlar</b>",
		"top.col.user":  "USER",
		"top.col.name":  "BUYRUQ",
		"top.none":      "<i>ko'rsatadigan narsa yo'q</i>",

		"help.top.short": "Eng ko'p CPU va xotira ishlatayotgan jarayonlar",
		"help.top": "Eng ko'p CPU ishlatayotgan 10 ta jarayonni (bir soniya davomida o'lchanadi) va eng ko'p xotira ishlatayotgan 10 tasini ko'rsatadi. CPU yoki RAM alertidan keyin foydali; bu alertlarning o'zida top 3 tasi allaqachon bor.\n\n" +
			"CPU%% bitta yadroga nisbatan, <code>top</code> dagidek: 4 yadroda band dastur 400 ko'rsatishi mumkin.\n\n" +
			"<b>Misol</b>\n<code>/top</code>: hozir serverni nima yuklayapti",
	})
}
