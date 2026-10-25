# Jadval va retention

`/schedule` backup **qachon** olinishini belgilaydi. `/retention` esa **nechtasi** saqlanishini.

## Jadval

```
/schedule 1 03:00
          │   │
          │   └── soat nechada: 03:00
          └────── necha kunda bir marta: har 1 kunda
```

| Buyruq | Ma'nosi |
|---|---|
| `/schedule 1 03:00` | Har kuni 03:00 da |
| `/schedule 2 03:00` | Kunora 03:00 da |
| `/schedule 7 04:00` | Haftada bir marta 04:00 da |
| `/schedule 30 04:00` | Taxminan oyda bir marta |
| `/schedule off` | Avtomatik backup o'chiq |

Kunlar soni 1 dan 365 gacha bo'lishi mumkin. `/backup` → **📅 Schedule** dagi tugmalar (Daily,
Every 3 days, Weekly, Off) faqat kunlar sonini o'zgartiradi, vaqt o'zgarmaydi.

> O'rnatilgandan keyin jadval **o'chiq** turadi. `/schedule` yubormaguningizcha backup faqat
> **Backup now** bosilganda olinadi.

Ma'lumotlari har kuni o'zgaradigan serverlar uchun (sayt, baza, yuklangan fayllar) `/schedule 1` eng
to'g'ri tanlov: deduplikatsiya tufayli kundalik backup faqat o'zgargan qismni qo'shadi.

### Qanday ishlaydi

1. Jadval o'rnatilganda vps-alert keyingi ishga tushish vaqtini, ya'ni hozirdan keyingi birinchi mos
   vaqtni hisoblaydi va `/var/lib/vps-alert/settings.json` ga yozadi.
2. Servis har 30 soniyada soatni tekshiradi.
3. Vaqt kelganda avval undan keyingi ishga tushish vaqtini saqlaydi, keyin backupni boshlaydi. Server
   backup paytida yiqilsa, yoqilgandan keyin o'sha backup qaytadan boshlanib ketmaydi.

Keyingi backup vaqti `/backup` menyusida (**⏭ Next**) va kunlik hisobotda ko'rsatiladi.

| Holat | Nima bo'ladi |
|---|---|
| Servis qayta ishga tushdi | Jadval davom etadi, chunki u diskda saqlanadi |
| Backup vaqtida server o'chiq edi | Server yoqilishi bilan bitta backup olinadi, keyin jadval odatdagidek davom etadi. Server bir hafta o'chiq tursa ham yetti emas, bitta backup olinadi |
| Rejadagi vaqtda qo'lda boshlangan backup hali ishlayapti | Rejadagi backup o'tkazib yuboriladi, keyingisi o'z vaqtida olinadi |
| **Backup now** bosildi | Jadvalga ta'sir qilmaydi |

### Vaqt zonasi

Vaqt **serverning** vaqt zonasi bo'yicha olinadi, ko'p VPS'lar esa UTC'da ishlaydi. Tekshirish va
o'zgartirish:

```sh
timedatectl
sudo timedatectl set-timezone Asia/Tashkent
sudo systemctl restart vps-alert
```

Shundan keyin `/schedule` ni qayta yuboring, shunda keyingi vaqt yangi zona bo'yicha hisoblanadi.

## Retention

```
/retention 7 4 6
           │ │ └── oylik:    oxirgi 6 oyning har biridan bittadan
           │ └──── haftalik: oxirgi 4 haftaning har biridan bittadan
           └────── kunlik:   oxirgi 7 kunning har biridan bittadan
```

Har bir backupdan keyin bu qoidalarning hech biriga to'g'ri kelmaydigan snapshot'lar o'chiriladi,
shuning uchun joy cheksiz o'sib bormaydi. Har bir kun, hafta yoki oy ichida eng yangi snapshot
saqlanadi.

### Misol

Backup har kuni olinadi, `/retention 7 4 6`, bugun 25-oktabr:

| Qoida | Saqlanadiganlari |
|---|---|
| 7 kunlik | 19, 20, 21, 22, 23, 24, 25-oktabr |
| 4 haftalik | har haftaning oxirgisi: 25, 19, 12, 5-oktabr |
| 6 oylik | har oyning oxirgisi: 25-okt, 30-sen, 31-avg, 31-iyul, 30-iyun, 31-may |

Bitta snapshot bir nechta qoidaga to'g'ri kelishi mumkin (25-oktabr uchala ro'yxatda ham bor),
shuning uchun 17 emas, taxminan 14 ta snapshot saqlanadi. Natijada oxirgi haftaning istalgan kuniga,
oxirgi oyning istalgan haftasiga va oxirgi yarim yilning istalgan oyiga qaytish mumkin.

Eski snapshot'lar muhim, chunki muammolar har doim darhol sezilmaydi. Agar jadval uch hafta oldin
o'chirilgan bo'lsa, barcha kunlik snapshot'larda u allaqachon yo'q. Haftalik yoki oylik snapshot'da
esa u hali bor.

### Joy

14 ta snapshot 14 ta to'liq nusxa degani emas. Deduplikatsiya tufayli kuniga 50 MB o'zgaradigan 3 GB'lik
server uchun hammasi taxminan 4–5 GB joy oladi.

### Boshqa variantlar

| Buyruq | Ma'nosi |
|---|---|
| `/retention 7 4 6` | Standart: bir hafta kunlik, bir oy haftalik, yarim yil oylik |
| `/retention 14 8 12` | Ko'proq tarix: ikki hafta, ikki oy, bir yil |
| `/retention 3 0 0` | Faqat oxirgi 3 kun, joy kam bo'lganda |
| `/retention 7 0 3` | Bir hafta kunlik va 3 ta oylik |

`0` qoidani o'chiradi, lekin uchala son birdaniga `0` bo'lishi mumkin emas.

Aniqroq aytganda, "kunlik 7" oxirgi 7 ta *snapshot olingan kun* degani. Shuning uchun backup har 3
kunda olinsa, `7` taxminan uch haftani qamrab oladi.

### Qamrovi

Retention faqat shu serverda vps-alert olgan snapshot'larga tegadi: ular `vps-alert` tegi va server
nomi bilan belgilangan. Shu repository'dagi boshqa restic snapshot'lariga tegilmaydi. Agar
`SERVER_NAME` ni o'zgartirsangiz, eski nom bilan olingan snapshot'lar endi avtomatik tozalanmaydi.
