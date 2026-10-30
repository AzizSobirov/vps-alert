# Telegram setup

vps-alert takes commands from exactly **one chat**, set by `ALLOWED_CHAT_ID` in
`/etc/vps-alert/vps-alert.env`. That chat can be a private chat with you or a group. Alerts go to the
same chat, or to another one such as a channel, see [Alerts in a channel](#alerts-in-a-channel).

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

## Alerts in a channel

Commands do not work in a Telegram channel: channel posts carry no sender, so the bot cannot check
who wrote them. To have alerts in a channel, send them there with `ALERT_CHAT_ID` and keep
`ALLOWED_CHAT_ID` for commands, a private chat or a group:

```sh
sudo nano /etc/vps-alert/vps-alert.env
#   ALLOWED_CHAT_ID=123456789          # commands and their answers
#   ALERT_CHAT_ID=-1001234567890       # alerts
#   ALLOWED_USER_IDS=123456789
sudo vps-alert test                    # sends a test message to both chats
sudo systemctl restart vps-alert
```

1. Add the bot to the channel as an administrator with **Post messages**.
2. To find the channel ID, forward a post from the channel to
   [@userinfobot](https://t.me/userinfobot), or post in the channel and look for `"chat":{"id":-100...`
   in `getUpdates` as for a group.

What goes where:

| `ALERT_CHAT_ID` | `ALLOWED_CHAT_ID` |
|---|---|
| Alerts, login alerts, the daily report, backup and restore results, the "started" message | Answers to commands, the menu, unauthorized access reports |

The **✅ It was me, trust this IP** button under a login alert works in the channel, but only for the
users in `ALLOWED_USER_IDS` (or, if it is empty, the owner of a private `ALLOWED_CHAT_ID`), since
anyone can read a channel. Commands sent in the alert chat are ignored. Leave `ALERT_CHAT_ID` empty
to send everything to `ALLOWED_CHAT_ID`.

## Switching between private chat and group

```sh
sudo nano /etc/vps-alert/vps-alert.env
#   ALLOWED_CHAT_ID=-1001234567890
#   ALLOWED_USER_IDS=123456789,987654321
sudo systemctl restart vps-alert
```

## Language and help

The bot speaks English, Uzbek and Russian. Send `/lang` and tap a language, or `/lang uz`. Messages,
buttons and the Telegram command menu switch at once. The default for a fresh install can be set
with `LANGUAGE=uz` (or `en`, `ru`) in the env file; `/lang` overrides it.

`/help` lists every command with an example, and `/help <command>` (e.g. `/help schedule`) explains
one in detail with a link to these docs. A command that needs arguments, sent without them, shows
its current setting and examples.

## Menu

Send `/menu` (or `/start`) to get a menu with buttons for every section: status, top processes,
backups, databases, services, websites, security, alerts and language. It is one message that
changes in place as you tap, so the chat does not fill up.

Common values are buttons too: schedule (daily, every 3 days, weekly), retention presets, mute for
30m / 2h / 1d, security switches. For values you have to type, like a folder or a website address,
the bot asks and you reply to its question. In a private chat just send the answer; in a group,
use Telegram's **Reply** on the question, so the bot knows which message is the answer.

Every command keeps working, so you can use whichever is quicker.

## Recommended BotFather settings

- `/setjoingroups` → **Disable** after adding the bot to your group, so strangers cannot add it to
  theirs.
- Keep the token secret. Anyone who has it can send messages as your bot and read updates sent to it.
  If it leaks, use `/revoke` in BotFather and put the new token in the env file.
