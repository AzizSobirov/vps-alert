# Bazalar

Ishlab turgan bazaning fayllarini oddiy nusxalash ularni yarim yozilgan holatda saqlab qolishi
mumkin. Shuning uchun vps-alert har bir bazani backupdan oldin izchil `.sql` faylga dump qiladi.

## Bazalarni tanlash

`/db` yuboring (yoki `/backup` dagi **🗄 Databases** tugmasini bosing). vps-alert quyidagilarni
qidiradi:

| Qayerda | Qanday topiladi |
|---|---|
| Serverdagi PostgreSQL | `psql` o'rnatilgan va `postgres` tizim foydalanuvchisi bor |
| Serverdagi MySQL / MariaDB | `mysqld` yoki `mariadbd` o'rnatilgan |
| Docker konteynerlar | image'i `postgres`, `postgis`, `timescale`, `pgvector`, `mysql`, `mariadb` yoki `percona` bo'lgan ishlab turgan konteynerlar |

U kira oladigan barcha bazalarni ko'rsatadi. Bazani bossangiz, u ✅ (backup qilinadi) va ⬜
(qilinmaydi) o'rtasida almashadi:

```
✅ postgres · shop
✅ docker:api-db · api
⬜ docker:api-db · test
✅ mysql · wordpress
[🔄 Rescan] [« Back]
```

Tizim bazalari (`postgres`, `mysql`, `sys`, `information_schema`, `performance_schema`) ko'rsatilmaydi.
Tanlangan, lekin endi topilmayotgan baza *(not found)* deb ko'rsatiladi, shunda uni ro'yxatdan olib
tashlash mumkin.

## Backup vaqtida

- Tanlangan har bir baza `/var/lib/vps-alert/dumps/` ichidagi alohida faylga dump qilinadi, masalan
  `postgres/shop.sql` yoki `docker-api-db/api.sql`. Ular bilan birga ularni tavsiflovchi
  `manifest.json` ham yoziladi.
- restic bu fayllarni backupning qolgan qismi bilan birga saqlaydi, keyin ular diskdan o'chiriladi.
- Muvaffaqiyatsiz dump ❌ bilan xabar qilinadi va boshqalarni to'xtatmaydi.

| Baza | Dump buyrug'i |
|---|---|
| PostgreSQL | `pg_dump` (oddiy SQL) |
| MySQL / MariaDB | `mysqldump` / `mariadb-dump --single-transaction --routines --triggers --events` |

## Parolsiz kirish

vps-alert baza parollarini so'ramaydi. U root'da allaqachon bor kirish huquqidan foydalanadi:

- **Serverdagi PostgreSQL**: `postgres` tizim foydalanuvchisi nomidan ishlaydi (peer authentication).
- **Serverdagi MySQL / MariaDB**: lokal socket orqali `root` bo'lib ulanadi (Debian va Ubuntu'da
  standart sozlama).
- **Docker**: klientni konteyner ichida, rasmiy image'lar o'z environment'ida saqlaydigan
  ma'lumotlar bilan ishga tushiradi: `POSTGRES_USER`, `MYSQL_ROOT_PASSWORD`, `MARIADB_ROOT_PASSWORD`
  yoki ularning `_FILE` variantlari.

Baza parollari bot orqali hech qachon kiritilmaydi, chunki ular chat tarixida qolib ketadi.

### Baza topilmasa

Skanerlash ogohlantirishlari ro'yxat ostida ko'rsatiladi. Ko'p uchraydigan sabablar:

- MySQL konteyneri `MYSQL_RANDOM_ROOT_PASSWORD` bilan ishga tushirilgan: o'qiladigan parol yo'q.
- Serverdagi MySQL root socket orqali emas, parol bilan kiradi.
- Baza masofada yoki boshqariladigan xizmatda (RDS, Supabase, ...): u bu serverda emas.

Bunday hollarda env faylga o'z dump buyrug'ingizni yozing. U `/paths add` bilan qo'shilgan papkaga
saqlashi kerak:

```sh
BACKUP_PRE_COMMAND=PGPASSWORD=secret pg_dump -h db.example.com -U app app > /var/backups/db/app.sql
```

Bu buyruq har backupdan oldin bajariladi va faqat env faylda yoziladi, bot orqali emas.

## Tiklash

`sudo vps-alert restore` → **A database** dumpni bazaga qaytaradi. Qarang: [tiklash](restore.md#baza).

## Ma'lum cheklovlar

- PostgreSQL rollari (foydalanuvchilari) `pg_dump` ga kirmaydi. Rollar yo'q yangi serverga tiklashda
  `role "..." does not exist` xatolari chiqadi. Avval foydalanuvchilarni yarating, keyin qayta
  tiklang.
- MongoDB, Redis va SQLite hozircha topilmaydi. Backup qilinayotgan papka ichidagi SQLite fayllari
  oddiy fayl sifatida nusxalanadi.
