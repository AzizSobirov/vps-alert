# Xavfsizlik kuzatuvi

[Login alertlari](alerts.md#login-alertlari)dan tashqari vps-alert hujum yoki istalmagan
o'zgarishning uchta belgisini kuzatadi: SSH'ga parol terib ko'rish (brute force), muhim fayllar
o'zgarishi va yangi ochilgan portlar.

Bu alertlarning hech birini `/mute` o'chirmaydi: login alertlari kabi ular yuklama emas, xavfsizlik
haqida.

## Muvaffaqiyatsiz SSH urinishlari

vps-alert sshd yozuvlarini journald'dan (u yerda bo'lmasa, `/var/log/auth.log` yoki
`/var/log/secure` dan) o'qiydi va muvaffaqiyatsiz kirishlarni sanaydi:

- mavjud foydalanuvchi uchun noto'g'ri parol,
- mavjud bo'lmagan foydalanuvchi bilan kirishga urinish,
- autentifikatsiya paytida uzilgan ulanish, masalan rad etilgan kalit, agar u haqida boshqa hech
  narsa yozilmagan bo'lsa.

sshd har bir urinish uchun bir nechta qator yozsa ham, urinish bir marta sanaladi.

Har 5 daqiqada oxirgi bir soat tekshiriladi. Urinishlar chegaraga (standart 100) yetsa, shunday xabar
keladi:

```
🚨 SSH brute force: my-server
Oxirgi bir soatda 1 240 ta muvaffaqiyatsiz kirish urinishi

Eng faol IP'lar:
  203.0.113.7: 830
  198.51.100.4: 310
Sinab ko'rilgan loginlar:
  root: 900
  admin: 120
```

Bunday alert soatiga ko'pi bilan bir marta keladi, servis qayta ishga tushganda ham.

| Buyruq | Ma'nosi |
|---|---|
| `/ssh` | Oxirgi 24 soatdagi muvaffaqiyatsiz urinishlar, eng faol IP'lar va loginlar bilan |
| `/ssh threshold 50` | Soatiga 50 ta urinishda alert |
| `/ssh off` · `/ssh on` | Alertni o'chirish yoki qayta yoqish (soatiga 100) |

SSH internetga ochiq bo'lgan serverga doim qandaydir urinishlar bo'ladi. Parol bilan kirish o'chiq
yoki parollar kuchli bo'lsa, ular zararsiz. Alert odatiy fon uchun emas, g'ayrioddiy to'lqinlar
uchun. Bir xil IP'lar qayta-qayta chiqsa, [fail2ban](https://github.com/fail2ban/fail2ban) o'rnatish
yoki SSH'ni faqat o'z IP'laringizdan ochish haqida o'ylab ko'ring.

## Muvaffaqiyatli kirishlar va SSH sozlamalari

`/ssh` oxirgi 24 soatdagi **muvaffaqiyatli** kirishlarni ham ko'rsatadi: foydalanuvchi, manzil, usul
va vaqt. Shuningdek, sshd parol va root bilan kirishga ruxsat beradimi, shuni ham ko'rsatadi:

```
✅ Oxirgi 24 soatdagi muvaffaqiyatli SSH kirishlar: 2
  aziz · 92.63.204.12 · kalit · 14:02
  root · 10.0.0.5 · parol ⚠️ · 03:40

⚙️ SSH sozlamalari
Parol bilan kirish: ruxsat etilgan ⚠️
Root kirishi: ruxsat etilgan, parol bilan ham ⚠️
```

Sozlamalar `sshd -T` dan olinadi, shuning uchun standart qiymatlar va `sshd_config.d/` dagi fayllar
ham hisobga olinadi.

Bunga ikkita alert qo'shilgan, `/mute` ularning hech birini o'chirmaydi:

- **Yangi manzildan birinchi kirish.** Manzil oldin hech qachon kirmagan bo'lsa, odatdagi 🔐 login
  xabariga 🆕 qatori qo'shiladi. Birinchi ishga tushishda vps-alert oxirgi 30 kunlik loglardan sizning
  odatiy manzillaringizni o'rganib oladi, ular belgilanmaydi. Ro'yxat
  `/var/lib/vps-alert/known-ips.json` da saqlanadi.
- **Parol bilan kirish.** Kimdir kalit o'rniga parol bilan kirsa, bir daqiqa ichida ⚠️ alert keladi.
  Parolni ataylab ishlatsangiz, `/ssh password off` bilan o'chiring.

### Faqat kalit bilan kirish

Har kuni minglab botlar parollarni sinab ko'radi. Parol bilan kirish o'chirilsa, ular umuman kira
olmaydi va muvaffaqiyatsiz urinishlar soni deyarli nolga tushadi.

1. SSH kalit bilan kira olishingizni tekshiring. O'z kompyuteringizda: `ssh-copy-id user@server-ip`,
   keyin `ssh user@server-ip` parol so'ramasdan kirishi kerak.
2. **Joriy SSH oynasini yopmang**, xato bo'lsa, undan qaytarasiz.
3. Serverda:

   ```sh
   printf "PasswordAuthentication no\nPermitRootLogin prohibit-password\n" | sudo tee /etc/ssh/sshd_config.d/10-keys-only.conf
   sudo sshd -t && sudo systemctl restart ssh
   ```

   RHEL asosidagi tizimlarda servis nomi `sshd`.
4. **Yangi** terminalda qayta kiring. Kirsa, ish tugadi: `/ssh` endi ikkala sozlamani ✅ deb
   ko'rsatadi. Kira olmasangiz, ochiq qolgan oynadan faylni o'chiring va ssh'ni qayta ishga tushiring.

Ba'zi bulut image'larida `PasswordAuthentication yes` alohida faylda yozilgan bo'ladi, masalan
`/etc/ssh/sshd_config.d/50-cloud-init.conf`. Fayllar nomi bo'yicha tartibda o'qiladi va birinchi
qiymat ustun turadi, shuning uchun yuqoridagi fayl nomi `10-` bilan boshlanadi.

## Ishonchli manzillar va foydalanuvchilar

`/whitelist` da kirishlari ovozsiz keladigan va hech qachon yangi deb belgilanmaydigan manzil, tarmoq
va foydalanuvchi nomlari saqlanadi (qarang: [alertlar](alerts.md#ishonchli-manzillar-va-foydalanuvchilar)).

```
/whitelist add 92.63.204.12     manzil
/whitelist add 10.0.0.0/24      tarmoq
/whitelist add deploy           foydalanuvchi, istalgan manzildan
/whitelist rm 92.63.204.12
```

### CI deploy'lar (GitHub Actions va boshqalar)

CI runner'lar har safar boshqa manzildan ulanadi, shuning uchun manzilni oq ro'yxatga qo'shish ish
bermaydi. GitHub'ning barcha diapazonlarini qo'shish esa ulardagi har qanday bulut serverni ishonchli
qilib qo'yadi. Deploy'lar uchun alohida foydalanuvchi oching va uni ishonchli qiling:

1. Faqat deploy uchun foydalanuvchi yarating, masalan `sudo adduser --disabled-password deploy`,
   unga deploy uchun kerakli huquqlarnigina bering va CI'ning ochiq kalitini
   `/home/deploy/.ssh/authorized_keys` ga qo'shing.
2. `/whitelist add deploy`

Shundan keyin deploy kirishlari ovozsiz keladi, yangi deb belgilanmaydi va ularning manzillari
ma'lum manzillar ro'yxatini to'ldirib yubormaydi. Bu foydalanuvchi parol bilan kirsa, parol alerti
baribir keladi, shuning uchun kalit ishlating.

## Muhim fayllar

Bu fayllar har daqiqada tekshiriladi:

| Nima | Fayllar |
|---|---|
| Foydalanuvchi va guruhlar | `/etc/passwd`, `/etc/group`, `/etc/shadow` |
| Admin huquqlari | `/etc/sudoers`, `/etc/sudoers.d/*` |
| SSH | `/etc/ssh/sshd_config`, `/etc/ssh/sshd_config.d/*`, `/root/.ssh/authorized_keys`, `/home/*/.ssh/authorized_keys` |
| Rejali vazifalar | `/etc/crontab`, `/etc/cron.d/*`, `/var/spool/cron/crontabs/*` |
| Servislar | `/etc/systemd/system/*.service` |
| Kutubxona kiritish | `/etc/ld.so.preload` |

Fayl yaratilsa, o'chirilsa, o'zgarsa, huquqlari yoki egasi o'zgarsa xabar qilinadi, muhim joylarda
tafsilotlar bilan:

```
🛡 Muhim fayllar o'zgardi: my-server

✏️ /etc/passwd o'zgardi
   👤 Yangi foydalanuvchi: mallory (uid 1002)
   🚨 Root huquqli foydalanuvchi (uid 0): toor
✏️ /etc/group o'zgardi
   👥 mallory sudo guruhiga qo'shildi
✏️ /root/.ssh/authorized_keys o'zgardi
   🔑 Yangi SSH kalit: SHA256:j4bhX0W+JGIpAlh1rFmC80zrN/pcjTWFFbLrkfl9kqU ssh-ed25519 laptop
🔐 /etc/sudoers huquqlari: 0440 0:0 → 0666 0:0
```

- Guruh o'zgarishlari admin huquqiga o'xshash huquq beradigan guruhlar uchun ko'rsatiladi: `sudo`,
  `wheel`, `admin` va `docker`.
- SSH kalitlari `ssh-keygen -lf` chiqaradigan SHA256 fingerprint bilan ko'rsatiladi, shuning uchun
  ularni o'z kalitlaringiz bilan solishtirishingiz mumkin.
- `/etc/shadow` uchun faqat o'zgargan akkauntlar nomi ko'rsatiladi. Parol hashlari hech qachon
  yuborilmaydi va saqlanmaydi: qaysi akkaunt o'zgarganini bilish uchun vps-alert har qatorning qisqa
  hashini saqlaydi.
- Fayl mazmuni hech qachon yuborilmaydi.

O'rnatilgandan keyingi birinchi tekshiruv faqat joriy holatni yozib oladi. vps-alert to'xtab turgan
paytdagi o'zgarishlar u qayta ishga tushganda xabar qilinadi. Holat
`/var/lib/vps-alert/filewatch.json` da saqlanadi.

| Buyruq | Ma'nosi |
|---|---|
| `/files` | Holat: kuzatilayotgan fayllar soni va oxirgi o'zgarish |
| `/files off` | To'xtatib turish, masalan rejali texnik ishlar paytida |
| `/files on` | Joriy holatdan qayta kuzatish: to'xtab turgan paytdagi o'zgarishlar xabar qilinmaydi |

## Ochiq portlar

Tarmoqda tinglayotgan yangi dastur buzilishning keng tarqalgan belgisi (backdoor, maynerning
boshqaruv porti) yoki noto'g'ri sozlama (internetga ochilib qolgan baza) bo'lishi mumkin.

vps-alert har daqiqada tinglayotgan TCP va UDP portlarni ro'yxatga oladi. Yangi port paydo bo'lib,
bir daqiqadan keyin ham ochiq tursa, shunday xabar keladi:

```
🔌 Yangi port ochildi: my-server

tcp 0.0.0.0:6379 (redis-server)
```

- Bir marta ko'rilgan portlar ma'lum bo'lib qoladi, shuning uchun portini yopib qayta ochadigan servis
  restarti alert bermaydi.
- Qisqa umrli socketlar va DNS hamda NTP klientlari ishlatadigan vaqtinchalik diapazondagi
  (32768–60999) UDP socketlar hisobga olinmaydi.
- `0.0.0.0` yoki `[::]` barcha tarmoq interfeyslaridan kirish mumkinligini, `127.0.0.1` yoki `[::1]`
  esa faqat serverning o'zidan kirish mumkinligini bildiradi.

| Buyruq | Ma'nosi |
|---|---|
| `/ports` | Hozirgi portlar va ularning dasturlari; 🆕 avval ko'rilmagan portlarni, *(yopilgan)* hozir tinglamayotgan ma'lum portlarni bildiradi |
| `/ports off` · `/ports on` | Alertni o'chirish yoki qayta yoqish |
