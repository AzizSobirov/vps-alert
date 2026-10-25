# Восстановление

Восстановление запускается **только из терминала сервера**. В боте нет команды восстановления, поэтому
даже с украденным телефоном или аккаунтом Telegram нельзя перезаписать ваши данные.

```sh
sudo vps-alert restore
```

Команда задаёт вопросы по шагам (на английском). Нажмите Enter, чтобы принять значение в `[скобках]`.
В любом выборе с номерами `q` выходит, ничего не меняя.

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

О каждом восстановлении сообщается в чат: **♻️ Восстановление на my-server, выполнил aziz**.

## Файлы и папки

Вы выбираете всё или одну из сохранённых папок и каталог назначения (по умолчанию
`/var/tmp/vps-alert-restore-<время>`).

Файлы **никогда не записываются поверх рабочей системы**. Под каталогом назначения они сохраняют полный
путь, так что `/var/www/shop` вернётся как `/var/tmp/vps-alert-restore-.../var/www/shop`. Сравните и
скопируйте нужное:

```sh
sudo rsync -a /var/tmp/vps-alert-restore-20261025-140000/var/www/shop/ /var/www/shop/
```

## База данных

Вы выбираете базу из снапшота, а затем один из вариантов:

- **Import into a database on this server**: загрузить в базу на этом сервере;
- **Only save the .sql file**: только сохранить `.sql` файл, чтобы посмотреть его или загрузить самому.

При загрузке:

- Если база ещё существует, предлагается имя `<имя>_restored`, и копия создаётся **рядом** с рабочей,
  ничего не теряется. Проверьте её, затем переключите приложение или перенесите данные.
- Чтобы заменить рабочую базу, введите её собственное имя. Затем имя нужно ввести ещё раз для
  подтверждения, и все её текущие данные будут удалены.
- Для баз в Docker можно выбрать другой контейнер, например если на новом сервере его имя изменилось.

После загрузки показываются ошибки, которые вывела база. `role "..." does not exist` означает, что
сначала нужно создать пользователей базы; см. [базы данных](databases.md#известные-ограничения).

## На новом сервере

Если старого сервера больше нет:

1. Настройте новый сервер и установите приложения (и контейнеры с базами), как раньше.
2. Установите vps-alert установщиком. Затем отредактируйте `/etc/vps-alert/vps-alert.env`, указав
   **те же настройки R2 и тот же `RESTIC_PASSWORD`**, что были на старом сервере, и перезапустите
   сервис.
3. Запустите `sudo vps-alert restore` и выберите **r2**.

Без старого `RESTIC_PASSWORD` бэкапы открыть нельзя. См. [пароль бэкапов](backup-password.md).

## Без vps-alert

Бэкапы — это обычные репозитории restic. Восстановить их может любой компьютер с restic и паролем:

```sh
export RESTIC_PASSWORD='...'
export RESTIC_REPOSITORY=s3:https://<account-id>.r2.cloudflarestorage.com/<bucket>/vps-alert
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=auto
# для локального репозитория: export RESTIC_REPOSITORY=/var/backups/vps-alert

restic snapshots                                         # список снапшотов
restic ls latest                                         # файлы в самом новом
restic restore latest --target /tmp/restore              # всё
restic restore latest --target /tmp/restore --include /var/www/shop          # одну папку
restic dump latest /var/lib/vps-alert/dumps/postgres/shop.sql > shop.sql     # дамп одной базы
```
