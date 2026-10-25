# Telegram setup

vps-alert talks to exactly **one chat**, set by `ALLOWED_CHAT_ID` in `/etc/vps-alert/vps-alert.env`.
That chat can be a private chat with you or a group.

## Private chat or group

| | Private chat | Group |
|---|---|---|
| `ALLOWED_CHAT_ID` | Your user ID, a positive number like `123456789` | The group ID, a negative number like `-1001234567890` |
| `ALLOWED_USER_IDS` | Optional: you are the only person in the chat | **Required in practice**: without it every group member can control the server |
| Who sees alerts | Only you | Everyone in the group |
| Good for | A personal server | A team, several admins, several servers |

### Private chat

1. Open your bot in Telegram and press **Start**. A bot cannot message you until you do.
2. Get your user ID from [@userinfobot](https://t.me/userinfobot).
3. Use that number as both the chat ID and the allowed user ID.

### Group

1. Add the bot to the group and send `/start` there.
2. Open `https://api.telegram.org/bot<TOKEN>/getUpdates` in a browser and find `"chat":{"id":-100...`.
   Do this **before** vps-alert is running; once it runs it consumes the updates and this page is empty.
3. Put the IDs of everyone allowed to give commands in `ALLOWED_USER_IDS`, separated by commas.

## Who can do what

- Commands are accepted only from `ALLOWED_CHAT_ID`, and when `ALLOWED_USER_IDS` is set, only from
  those users.
- Anyone else is ignored. If the bot is added to another group, it leaves immediately.
- Each rejected attempt is reported to your chat as **⚠️ Unauthorized access attempt**, at most once
  per person per hour.
- Commands sent while the server was offline are dropped if they are more than 5 minutes old, so a
  "Backup now" pressed yesterday does not run when the server comes back.

## Several servers

One bot token can only be used by one server at a time: Telegram delivers each update to a single
receiver, and a second server gets `Conflict: terminated by other getUpdates request`.

So create **one bot per server** and add them all to the same group. Alerts from every server then
land in one place. To address one bot in a group with several, add its username to the command:

```
/status@shop_server_bot
```

A plain `/status` is answered by every bot in the group.

## Switching between private chat and group

```sh
sudo nano /etc/vps-alert/vps-alert.env
#   ALLOWED_CHAT_ID=-1001234567890
#   ALLOWED_USER_IDS=123456789,987654321
sudo systemctl restart vps-alert
```

## Recommended BotFather settings

- `/setjoingroups` → **Disable** after adding the bot to your group, so strangers cannot add it to
  theirs.
- Keep the token secret. Anyone who has it can send messages as your bot and read updates sent to it.
  If it leaks, use `/revoke` in BotFather and put the new token in the env file.
