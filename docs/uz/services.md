# Servislar va saytlar

vps-alert serverdagi biror narsa ishlamay qolganini aytib turadi: systemd servisi, Docker konteyneri
yoki server xizmat ko'rsatadigan sayt yoki API. Bu alertlar ham resurs alertlari kabi `/mute` bilan
o'chiriladi.

## Servis va konteynerlar

`/services` yuboring. Bot ishlab turgan systemd servislari va Docker konteynerlarini 🟢 ishlayapti
yoki 🔴 to'xtagan belgisi bilan ko'rsatadi. Kuzatiladiganlarini bosing:

```
✅ 🟢 nginx
✅ 🟢 postgresql@16-main
⬜ 🟢 ssh
✅ 🔴 🐳 api (to'xtagan)
[🔄 Rescan]
```

Operatsion tizimning ichki servislari (`systemd-*`, `getty`, `dbus`, `cron`, ...) ko'rsatilmaydi.
Kuzatilayotgan, lekin endi ishlamayotgan servis ro'yxatda qoladi, uni belgidan chiqarish mumkin.

Har 30 soniyada har bir kuzatilayotgan servis tekshiriladi:

| Hodisa | Xabar |
|---|---|
| To'xtagan yoki xato bilan tushgan, keyingi tekshiruvda ham ishlamayapti | 🔴 **nginx** servisi xato bilan to'xtagan |
| Konteyner to'xtagan | 🔴 **api** konteyneri to'xtagan (chiqish kodi 137) |
| Yana ishlayapti | ✅ **nginx** servisi yana ishlayapti (4m to'xtab turdi) |
| 10 daqiqada 3 va undan ko'p restart | 🔁 **api** konteyneri qayta-qayta ishga tushyapti: 10 daqiqada 4 marta |
| Serverdan o'chirilgan | ⚠️ **old** servisi topilmadi. Agar u o'chirilgan bo'lsa, /services da belgisini olib tashlang. |

Oddiy restart bir necha soniya davom etadi, shuning uchun alert faqat servis keyingi tekshiruvda ham,
ya'ni 30 soniyadan keyin ham ishlamayotgan bo'lsa keladi. Yiqilib avtomatik qayta ishga tushadigan
servis (`Restart=always`, Docker `restart: always`) har tekshiruvda sog'lom ko'rinadi. Bunday holatni
qayta-qayta ishga tushish alerti ushlaydi.

## Saytlar va API'lar

```
/http add https://example.com
/http add https://api.example.com/health
/http rm https://example.com
/http                                    barcha manzillar va holati
```

20 tagacha manzil. Har biri har daqiqada 10 soniyalik kutish bilan ochiladi, redirect'lar kuzatiladi.

| Hodisa | Xabar |
|---|---|
| Ketma-ket ikki marta ishlamadi | 🔴 **https://example.com** ishlamayapti: 502 Bad Gateway |
| Qayta ishladi | ✅ **https://example.com** yana ishlayapti (6m ishlamay turdi) |
| Ketma-ket uch marta 5 soniyadan sekin | 🐢 **https://example.com** sekin ishlayapti: 6.2s da javob berdi |

Ulanish xatosi, javob kelmasligi, DNS yoki SSL sertifikat xatosi yoki 400 va undan yuqori HTTP status
ishlamaslik deb hisoblanadi. Sekinlik haqidagi ogohlantirish bir marta keladi va sayt yana tez
ishlamaguncha takrorlanmaydi.

Tekshiruvlar serverning o'zidan bajariladi. Shuning uchun ular yiqilgan ilovani yoki buzilgan web
server sozlamasini ushlaydi, lekin foydalanuvchilar bilan server orasidagi tarmoq uzilishini
ushlamaydi. Buning uchun tashqi monitoringdan ham foydalaning, qarang: [heartbeat](heartbeat.md).
