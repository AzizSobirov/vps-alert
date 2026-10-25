# Alertlar va hisobotlar

## Resurs alertlari

vps-alert har 30 soniyada CPU, RAM, disk va inode bandligini tekshiradi.

| Sozlama | Standart | O'zgartirish |
|---|---|---|
| CPU | 90% | `/alerts cpu 90` |
| RAM | 90% | `/alerts mem 90` |
| Disk | 85% | `/alerts disk 85` |
| Inode | 85% | `/alerts inode 85` |
| Davom etishi kerak | 5 daqiqa | `/alerts duration 5` |
| Qayta eslatish | 60 daqiqa | `/alerts cooldown 60` |

Joriy qiymatlarni ko'rish uchun `/alerts` yuboring. Chegarani `0` qilsangiz, u o'chadi.

### Alert qachon chiqadi va qachon yo'qoladi

1. Qiymat chegaradan oshadi. Hali hech narsa yuborilmaydi.
2. Agar u to'liq **davomiylik** (5 daqiqa) davomida yuqori tursa, **🔴 CPU: 95% (threshold 90%)**
   keladi. Undan oldin pasayib ketgan qisqa sakrash hech qachon alert bermaydi.
3. Qiymat yuqoriligicha qolsa, har **cooldown** (60 daqiqa) da bir marta **🔴 CPU still high**
   eslatmasi keladi.
4. Qiymat chegaradan **5 punkt pastga** tushganda **✅ CPU back to normal** keladi. Bu 5 punktlik
   zaxira chegara atrofida tebranib turgan qiymat ketma-ket alert yuborishining oldini oladi.

Har bir disk alohida tekshiriladi. Vaqtinchalik va virtual fayl tizimlari (tmpfs, Docker overlay,
snap) hisobga olinmaydi.

**Inode** diskdagi fayllar uchun joylar. Diskda bo'sh joy bo'lsa ham inode'lar tugab qolishi mumkin,
odatda millionlab mayda kesh yoki sessiya fayllari sabab bo'ladi. Unda diskka hech narsa yozib
bo'lmaydi, shuning uchun u ham disk joyi kabi tekshiriladi.

## Alertlarni vaqtincha o'chirish

```
/mute 30m      /mute 2h      /mute 1d      /unmute
```

`/mute` resurs alertlarini 7 kungacha o'chirib turadi, masalan texnik ishlar paytida. Login
alertlari, backup hisobotlari va kunlik hisobot o'chmaydi. Alertlar qachon qayta yoqilishini
`/status` ko'rsatadi.

## Login alertlari

Har bir SSH kirishda shunday xabar keladi:

```
🔐 Login on my-server

👤 User: root
🌐 From: 203.0.113.7
🔧 Service: sshd
🕓 2026-10-25 14:03:11 +05
```

Ularni PAM yuboradi: installer `/etc/pam.d/sshd` fayliga bitta qator qo'shadi. Natijada:

- Alertlar vps-alert servisi to'xtatilgan bo'lsa ham keladi.
- Qator `optional` deb belgilangan va hook darhol qaytadi, shuning uchun vps-alert buzilsa yoki
  sekinlashsa ham login hech qachon kechikmaydi va bloklanmaydi.
- Kalit bilan ham, parol bilan ham kirishlar xabar qilinadi.

O'zingizning kirishlaringiz uchun alert kelmasligi uchun IP yoki tarmoqlaringizni env faylga yozing
va servisni qayta ishga tushiring:

```sh
LOGIN_IP_WHITELIST=203.0.113.7,198.51.100.0/24
```

Login alertlarini butunlay o'chirish uchun `/etc/pam.d/sshd` dan `vps-alert` qatorini o'chiring.

Muvaffaqiyatsiz kirish urinishlari hozircha xabar qilinmaydi.

## Kunlik hisobot

Har kuni soat 09:00 da (server vaqti bilan) joriy holat, oxirgi backup natijasi va keyingi backup
vaqti keladi.

```
/report 08:30     vaqtni o'zgartirish
/report off       o'chirish
```

## Xizmat xabarlari

- **🟢 vps-alert started** servis har ishga tushganda keladi. Bu server qayta yoqilganini ham
  bildiradi.
- **⚠️ Unauthorized access attempt** ruxsat berilgan chat yoki foydalanuvchilardan tashqaridagi
  kimdir botdan foydalanmoqchi bo'lganini bildiradi. Qarang: [Telegram sozlash](telegram.md#kim-nima-qila-oladi).
