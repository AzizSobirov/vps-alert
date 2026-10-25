# Backup paroli (`RESTIC_PASSWORD`)

`RESTIC_PASSWORD` barcha backuplarni shifrlaydigan kalit. Backup local diskda bo'ladimi yoki R2'da,
uni faqat shu parolni biladigan odam ocha oladi.

## Nimani himoya qiladi

restic biror narsani yozishdan oldin quyidagilarni shifrlaydi:

- fayllar mazmuni,
- fayl va papka nomlari,
- hajmlar, sanalar va baza dumplari.

R2 bucket'ingizga qaragan odam, u Cloudflare xodimi bo'ladimi yoki R2 kalitlaringizni o'g'irlagan
odammi, faqat tasodifiy baytlarni ko'radi. U hatto `shop.sql` degan fayl borligini ham bila olmaydi.

Texnik tomondan repository ma'lumotlarni shifrlaydigan tasodifiy master kalitga ega (AES-256,
o'zgartirilishdan himoya bilan). Sizning parolingiz shu master kalitni ochadi.

## Parol qayerdan keladi

Installer 40 belgili tasodifiy parol yaratadi, uni **bir marta** ekranga chiqaradi va
`/etc/vps-alert/vps-alert.env` ga yozadi (faqat root o'qiy oladi). Uni qayta ko'rish:

```sh
sudo grep RESTIC_PASSWORD /etc/vps-alert/vps-alert.env
```

O'rnatgandan keyin darhol parolni password manager'ga (Bitwarden, 1Password, KeePass, ...) saqlang.

## Parol yo'qolsa, backuplar ham yo'qoladi

Tiklash yo'li yo'q. Parolsiz backuplarni na restic, na vps-alert, na Cloudflare ocha oladi.

Eng xavfli holat: server o'ladi, parol esa faqat uning env faylida edi. R2'dagi backuplar butun,
lekin ularni o'qib bo'lmaydi.

Yangi serverga tiklash uchun kerak bo'ladi:

1. `RESTIC_PASSWORD`,
2. R2 sozlamalari (`R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `R2_PREFIX`),
3. restic (yoki vps-alert).

Ularning hammasini password manager'da saqlang.

## Nimadan himoya qilmaydi

| Holat | Himoya bormi? |
|---|---|
| R2 kalitlari sizib chiqdi | ✅ Ma'lumotni o'qib bo'lmaydi |
| Backup diski yoki bucket o'g'irlandi | ✅ |
| Serverning o'zi buzildi (hujumchi root oldi) | ❌ Parol ham, R2 kalitlari ham serverda turibdi |

Root olgan hujumchi backuplarni **o'chirib** ham yuborishi mumkin, chunki serverda yozish huquqi bor
R2 kalitlari turibdi. vps-alert'da bunga qarshi himoya hozircha yo'q. Bucket'dagi obyektlarni qulflash
ham oddiy yechim emas: u avtomatik tozalashni ham eski snapshot'larni o'chirishdan to'xtatib qo'yadi.
Bu muammo hal qilinmaguncha eng xavfsiz qo'shimcha nusxa boshqa kompyuter serverdan o'zi tortib oladigan,
server hech qachon ko'rmaydigan kalitlar bilan saqlanadigan nusxa hisoblanadi.

## Qoidalar

- **Har bir serverda boshqa parol ishlating.** Installer buni avtomatik qiladi.
- **Backuplar bor bo'lsa, env fayldagi `RESTIC_PASSWORD` ni shunchaki o'zgartirmang.** Repository
  eski parol bilan qulflangan, shuning uchun backuplar `wrong password or no key found` xatosi bilan
  to'xtaydi. Parolni pastda yozilganidek almashtiring.
- Local va R2 repository'lari bir xil parolni ishlatadi, lekin ularning har biri o'z master kalitiga ega.

## Parolni almashtirish

Avval har bir repository'da, keyin env faylda almashtiring:

```sh
sudo -s
set -a; . /etc/vps-alert/vps-alert.env; set +a     # joriy parolni yuklaydi

# local repository
RESTIC_REPOSITORY=$BACKUP_LOCAL_REPO restic key passwd

# R2 repository, agar ishlatilsa
RESTIC_REPOSITORY=s3:https://$R2_ACCOUNT_ID.r2.cloudflarestorage.com/$R2_BUCKET/$R2_PREFIX \
AWS_ACCESS_KEY_ID=$R2_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$R2_SECRET_ACCESS_KEY AWS_DEFAULT_REGION=auto \
restic key passwd
```

`restic key passwd` yangi parolni ikki marta so'raydi. Ikkala repository uchun bir xil yangi parolni
kiriting, keyin:

```sh
nano /etc/vps-alert/vps-alert.env      # RESTIC_PASSWORD=<yangi parol>
systemctl restart vps-alert
```

Faqat master kalitning qulfi o'zgaradi, ma'lumotlar qayta shifrlanmaydi, shuning uchun bu bir necha
soniya oladi. Yangi parolni password manager'ga saqlang va tekshirish uchun **Backup now** ni bosing.

## Parol umuman yozilmagan bo'lsa

Agar `RESTIC_PASSWORD=` bo'sh bo'lsa (masalan, env faylni qo'lda yozgan bo'lsangiz), hali bitta ham
backup olinmagan va `/backup` menyusida **⚠️ RESTIC_PASSWORD is not set** turgan bo'ladi. Parol
yarating va servisni qayta ishga tushiring:

```sh
openssl rand -base64 30
sudo nano /etc/vps-alert/vps-alert.env      # RESTIC_PASSWORD=<shu qiymat>
sudo systemctl restart vps-alert
```
