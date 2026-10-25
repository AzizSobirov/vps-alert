# Services and websites

vps-alert can tell you when something on the server stops working: a systemd service, a Docker
container, or a website or API it serves. These alerts are silenced by `/mute`, like resource alerts.

## Services and containers

Send `/services`. The bot lists the running systemd services and Docker containers, with
🟢 running or 🔴 stopped. Tap the ones to watch:

```
✅ 🟢 nginx
✅ 🟢 postgresql@16-main
⬜ 🟢 ssh
✅ 🔴 🐳 api (exited)
[🔄 Rescan]
```

Operating system internals (`systemd-*`, `getty`, `dbus`, `cron`, ...) are not offered. A watched
service that is no longer running stays in the list so you can deselect it.

Every 30 seconds each watched service is checked:

| Event | Message |
|---|---|
| Stopped or failed, still down on the next check | 🔴 Service **nginx** is failed |
| Container stopped | 🔴 Container **api** is exited (exit code 137) |
| Running again | ✅ Service **nginx** is running again (down 4m) |
| 3 or more restarts within 10 minutes | 🔁 Container **api** keeps restarting: 4 restarts in 10 minutes |
| Removed from the server | ⚠️ Service **old** was not found. If it was removed, deselect it in /services. |

A normal restart takes a few seconds, so an alert is sent only when the service is still down on the
next check, 30 seconds later. A service that crashes and is restarted automatically
(`Restart=always`, Docker `restart: always`) looks healthy at each check; the restart-loop alert
catches that case.

## Websites and APIs

```
/http add https://example.com
/http add https://api.example.com/health
/http rm https://example.com
/http                                    every address with its status
```

Up to 20 addresses. Each one is opened every minute with a 10-second timeout, following redirects.

| Event | Message |
|---|---|
| Fails twice in a row | 🔴 **https://example.com** is down: 502 Bad Gateway |
| Back | ✅ **https://example.com** is up again (down 6m) |
| Slower than 5 seconds three times in a row | 🐢 **https://example.com** is slow: answered in 6.2s |

A failure is a connection error, a timeout, a DNS or SSL certificate error, or an HTTP status of 400
or above. The slow warning is sent once and is not repeated until the site has been fast again.

The checks run from the server itself, so they catch a crashed app or a broken web server
configuration, but not a network outage between your users and the server. For that, also use an
external monitor; see [heartbeat](heartbeat.md).
