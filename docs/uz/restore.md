# Tiklash (restore)

Restore **faqat server terminalidan** ishlaydi. Botda restore buyrug'i yo'q, shuning uchun telefoningiz
yoki Telegram akkauntingiz o'g'irlansa ham ma'lumotlaringiz ustidan yozib bo'lmaydi.

```sh
sudo vps-alert restore
```

Buyruq bosqichma-bosqich savol beradi. `[qavs]` ichidagi qiymatni qabul qilish uchun Enter bosing.
Raqamli tanlovlarda `q` hech narsani o'zgartirmasdan chiqadi.

```
Restore from:
   1) local  /var/backups/vps-alert
   2) r2     my-backups/vps-alert
Choose [1]:

Snapshot (newest first):
   1) a1b2c3d4  2026-10-25 03:00    3.2 GiB
   2) e5f6a7b8  2026-10-24 03:00    3.1 GiB
Choose [1]:

What to restore:
   1) Files and folders
   2) A database
Choose [1]:
```

Har bir restore chatga xabar qilinadi: **♻️ Restore on my-server by aziz**.

## Fayl va papkalar

Hammasini yoki backup qilingan papkalardan birini tanlaysiz, keyin qayerga tiklashni (standart:
`/var/tmp/vps-alert-restore-<vaqt>`).

Fayllar **ishlab turgan tizim ustiga hech qachon yozilmaydi**. Ular tanlangan papka ichida to'liq yo'li
bilan tiklanadi: `/var/www/shop` → `/var/tmp/vps-alert-restore-.../var/www/shop`. Solishtiring va
keraklisini ko'chiring:

```sh
sudo rsync -a /var/tmp/vps-alert-restore-20261025-140000/var/www/shop/ /var/www/shop/
```

## Baza

Snapshot'dan bazani tanlaysiz, keyin ikki yo'ldan birini:

- **Import into a database on this server**: serverdagi bazaga import qilish;
- **Only save the .sql file**: faqat `.sql` faylini saqlash, uni ko'rib chiqish yoki o'zingiz import
  qilish uchun.

Import qilganda:

- Baza hali mavjud bo'lsa, taklif qilinadigan nom `<nom>_restored` bo'ladi. Nusxa ishlab turgan baza
  **yonida** yaratiladi va hech narsa yo'qolmaydi. Uni tekshiring, keyin ilovangizni unga o'tkazing
  yoki ma'lumotlarni ko'chiring.
- Ishlab turgan bazani almashtirish uchun uning o'z nomini yozing. Keyin tasdiqlash uchun nomni yana
  bir marta yozish kerak bo'ladi va uning barcha joriy ma'lumotlari o'chiriladi.
- Docker bazalari uchun boshqa konteynerni tanlash mumkin, masalan yangi serverda konteyner nomi
  o'zgargan bo'lsa.

Importdan keyin baza chiqargan xatolar ko'rsatiladi. `role "..." does not exist` xatosi baza
foydalanuvchilarini avval yaratish kerakligini bildiradi. Qarang: [bazalar](databases.md#malum-cheklovlar).

## Yangi serverda

Eski server yo'qolgan bo'lsa:

1. Yangi serverni sozlang va ilovalaringizni (hamda baza konteynerlarini) avvalgidek o'rnating.
2. vps-alert'ni installer bilan o'rnating. Keyin `/etc/vps-alert/vps-alert.env` ni tahrirlab, eski
   serverdagi **xuddi o'sha R2 sozlamalari va o'sha `RESTIC_PASSWORD`** ni yozing va servisni qayta
   ishga tushiring.
3. `sudo vps-alert restore` ni ishga tushiring va **r2** ni tanlang.

Eski `RESTIC_PASSWORD` siz backuplarni ochib bo'lmaydi. Qarang: [backup paroli](backup-password.md).

## vps-alert'siz

Backuplar oddiy restic repository'lari. restic va parol bor istalgan kompyuter ularni tiklay oladi:

```sh
export RESTIC_PASSWORD='...'
export RESTIC_REPOSITORY=s3:https://<account-id>.r2.cloudflarestorage.com/<bucket>/vps-alert
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=auto
# local repository uchun: export RESTIC_REPOSITORY=/var/backups/vps-alert

restic snapshots                                         # snapshot'lar ro'yxati
restic ls latest                                         # eng yangisidagi fayllar
restic restore latest --target /tmp/restore              # hammasini
restic restore latest --target /tmp/restore --include /var/www/shop          # bitta papkani
restic dump latest /var/lib/vps-alert/dumps/postgres/shop.sql > shop.sql     # bitta baza dumpini
```
