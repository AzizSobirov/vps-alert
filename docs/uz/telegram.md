# Telegram sozlash

vps-alert faqat **bitta chat** bilan ishlaydi. U `/etc/vps-alert/vps-alert.env` faylidagi
`ALLOWED_CHAT_ID` bilan belgilanadi va shaxsiy chat ham, guruh ham bo'lishi mumkin.

## Shaxsiy chat yoki guruh

| | Shaxsiy chat | Guruh |
|---|---|---|
| `ALLOWED_CHAT_ID` | Sizning user ID'ingiz, musbat son, masalan `123456789` | Guruh ID'si, manfiy son, masalan `-1001234567890` |
| `ALLOWED_USER_IDS` | Ixtiyoriy: chatda faqat siz bor | **Amalda majburiy**: usiz guruhdagi har bir a'zo serverni boshqara oladi |
| Alertlarni kim ko'radi | Faqat siz | Guruhdagi hamma |
| Kimga mos | Shaxsiy server | Jamoa, bir nechta admin, bir nechta server |

### Shaxsiy chat

1. Botingizni Telegram'da oching va **Start** bosing. Shundan oldin bot sizga yoza olmaydi.
2. User ID'ingizni [@userinfobot](https://t.me/userinfobot) dan oling.
3. Shu sonni chat ID sifatida ham, ruxsat berilgan user ID sifatida ham yozing.

### Guruh

1. Botni guruhga qo'shing va guruhda `/start` yuboring.
2. Brauzerda `https://api.telegram.org/bot<TOKEN>/getUpdates` ni oching va `"chat":{"id":-100...`
   qismini toping. Buni vps-alert ishga tushishidan **oldin** qiling: u ishlay boshlagach, xabarlarni
   o'zi qabul qiladi va bu sahifa bo'sh chiqadi.
3. Buyruq berishga ruxsat berilgan barcha odamlarning ID'larini vergul bilan ajratib
   `ALLOWED_USER_IDS` ga yozing.

## Kim nima qila oladi

- Buyruqlar faqat `ALLOWED_CHAT_ID` dan qabul qilinadi. `ALLOWED_USER_IDS` berilgan bo'lsa, faqat
  o'sha foydalanuvchilardan.
- Boshqa hamma e'tiborsiz qoldiriladi. Botni boshqa guruhga qo'shishsa, u darhol chiqib ketadi.
- Har bir rad etilgan urinish chatingizga **⚠️ Unauthorized access attempt** deb xabar qilinadi, bir
  odam uchun soatiga ko'pi bilan bir marta.
- Server o'chiq paytda yuborilgan buyruqlar 5 daqiqadan eski bo'lsa bajarilmaydi. Masalan, kecha
  bosilgan "Backup now" server yoqilganda ishga tushib ketmaydi.

## Bir nechta server

Bitta bot tokenini bir vaqtda faqat bitta server ishlata oladi: Telegram har bir xabarni bitta
qabul qiluvchiga beradi, ikkinchi server esa `Conflict: terminated by other getUpdates request`
xatosini oladi.

Shuning uchun **har bir serverga alohida bot** yarating va hammasini bitta guruhga qo'shing. Shunda
barcha serverlarning alertlari bir joyda turadi. Bir nechta bot bo'lgan guruhda buyruqni aniq bitta
botga yuborish uchun uning username'ini qo'shing:

```
/status@shop_server_bot
```

Oddiy `/status` ga guruhdagi barcha botlar javob beradi.

## Shaxsiy chat va guruh o'rtasida almashtirish

```sh
sudo nano /etc/vps-alert/vps-alert.env
#   ALLOWED_CHAT_ID=-1001234567890
#   ALLOWED_USER_IDS=123456789,987654321
sudo systemctl restart vps-alert
```

## Til va yordam

Bot inglizcha, o'zbekcha va ruscha gapiradi. `/lang` yuborib tilni tanlang yoki `/lang uz` deb
yozing. Xabarlar, tugmalar va Telegram'dagi buyruqlar menyusi darhol almashadi. Yangi o'rnatish
uchun standart tilni env faylda `LANGUAGE=uz` (yoki `en`, `ru`) bilan berish mumkin, `/lang` esa
uning ustidan yozadi.

`/help` barcha buyruqlarni misol bilan ko'rsatadi, `/help <buyruq>` (masalan `/help schedule`) esa
bittasini batafsil va shu hujjatlarga havola bilan tushuntiradi. Argument talab qiladigan buyruq
argumentsiz yuborilsa, joriy sozlama va misollar ko'rsatiladi.

## BotFather'da tavsiya etilgan sozlamalar

- Botni guruhingizga qo'shgandan keyin `/setjoingroups` → **Disable** qiling, shunda begonalar uni
  o'z guruhlariga qo'sha olmaydi.
- Tokenni sir saqlang. Token kimda bo'lsa, u botingiz nomidan xabar yubora oladi va unga kelgan
  xabarlarni o'qiy oladi. Token sizib chiqsa, BotFather'da `/revoke` qiling va yangi tokenni env
  faylga yozing.
