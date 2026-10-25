# Backup qanday ishlaydi

vps-alert fayllarni o'zi nusxalamaydi. U keng ishlatiladigan [restic](https://restic.net) backup
dasturini boshqaradi va uning ustiga jadval, baza dumplari va Telegram orqali boshqaruvni qo'shadi.

## Sozlash

```
/paths add /var/www        nimani backup qilish (har bir papka uchun takrorlang)
/db                        qaysi bazalarni dump qilish, qarang: databases.md
/backup → 🎯 Target        qayerga: Local, R2 yoki Both
/schedule 1 03:00          qachon, qarang: schedule-and-retention.md
/retention 7 4 6           nechtasini saqlash
```

Sozlamalar `/var/lib/vps-alert/settings.json` da saqlanadi va servis qayta ishga tushganda
yo'qolmaydi.

## Backup vaqtida nima bo'ladi

```
1. Tekshiruv       restic o'rnatilganmi, RESTIC_PASSWORD bormi, backup qiladigan narsa bormi
2. Pre-command     env fayldagi BACKUP_PRE_COMMAND, agar berilgan bo'lsa
3. Bazalar         tanlangan har bir baza alohida .sql faylga dump qilinadi
4. Har bir target uchun (local va/yoki R2):
     a. local: diskda kamida 1 GiB bo'sh joy bormi?
     b. faqat birinchi marta: repository yaratiladi (restic init)
     c. restic backup: o'zgargan qismlar shifrlanib saqlanadi
     d. restic forget --prune: retention'ga to'g'ri kelmaydigan snapshot'lar o'chiriladi
5. Tozalash        .sql dumplar diskdan o'chiriladi
6. Hisobot         natija chatga yuboriladi
```

Bir vaqtda faqat bitta backup ishlaydi. Backup ishlayotganda **Backup now** bossangiz, bot buni
aytadi.

## Hisobot

```
✅ Backup finished: my-server (scheduled)

🗄 Databases: 2/2 dumped
✅ postgres · shop: 120.0 MiB
✅ docker:api-db · api: 300.0 MiB

✅ local: snapshot a1b2c3d4, +12.4 MiB new, 3.2 GiB total, 48 new files
✅ r2: snapshot e5f6a7b8, +12.4 MiB new, 3.2 GiB total, 48 new files

⏱ 1m12s
```

- **+12.4 MiB new**: oldingi snapshot'larning hech birida bo'lmagan ma'lumot, ya'ni shu backup
  haqiqatda qo'shgan hajm.
- **3.2 GiB total**: backup qilingan hamma narsaning hajmi.
- Muvaffaqiyatsiz baza dumpi yoki target ❌ va xato matni bilan ko'rsatiladi, qolganlari baribir
  bajariladi.
- **⚠️ some files could not be read**: snapshot yaratildi, lekin ba'zi fayllarni o'qib bo'lmadi
  (odatda backup paytida o'chirilgan fayllar).

## Backuplar qayerga boradi

| Target | Joy |
|---|---|
| Local | `/var/backups/vps-alert` (`BACKUP_LOCAL_REPO`) |
| R2 | `<bucket>/vps-alert/` (`R2_BUCKET`, `R2_PREFIX`) |

Local backup o'chirilgan yoki buzilgan fayllardan himoya qiladi. Lekin serverni yoki uning diskini
yo'qotishdan **himoya qilmaydi**, chunki u o'sha diskda turadi. Buning uchun backuplarni R2'ga ham
yuboring va **Both** ni tanlang.

### Cloudflare R2'ni sozlash

1. Cloudflare dashboard'da **R2** ni oching va bucket yarating, masalan `my-backups`.
2. **Manage R2 API Tokens** → **Create API token**: *Object Read & Write* huquqi bilan, faqat shu
   bucket uchun.
3. Qiymatlarni `/etc/vps-alert/vps-alert.env` ga yozing:

   ```sh
   R2_ACCOUNT_ID=...          # R2 bosh sahifasida ko'rsatilgan
   R2_ACCESS_KEY_ID=...
   R2_SECRET_ACCESS_KEY=...
   R2_BUCKET=my-backups
   ```

4. `sudo systemctl restart vps-alert`, keyin `/backup` → **🎯 Target** → **Both**.

Har bir serverning `R2_PREFIX` i har xil bo'lsa, bir nechta server bitta bucket'dan foydalanishi
mumkin.

## Shifrlash va deduplikatsiya

- Hamma narsa serverdan chiqishidan oldin `RESTIC_PASSWORD` bilan shifrlanadi: fayl mazmuni, fayl
  nomlari, hajmlar. Bucket'ga kirish huquqi bor odam ham uni o'qiy olmaydi. Qarang:
  [backup paroli](backup-password.md).
- Fayllar bo'laklarga ajratiladi va har bir bo'lak faqat bir marta saqlanadi. Kuniga 50 MB
  o'zgaradigan 3 GB'lik sayt har backupda 3 GB emas, taxminan 50 MB qo'shadi.

## Cheklovlar

- Backup butun tizimni emas, siz tanlagan papka va bazalarni qamrab oladi. Serverni qayta tiklash
  uchun OS va ilovalarni qayta o'rnatasiz, keyin ma'lumotlarni restore qilasiz.
- Restore bot orqali emas, faqat server terminalidan qilinadi. Qarang: [tiklash](restore.md).
