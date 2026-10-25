# Heartbeat

A server that is down cannot report that it is down. If it loses power, hangs or is suspended by the
host, vps-alert stops with it, and the chat simply goes quiet. Silence looks the same whether
everything is fine or the server is dead.

The heartbeat turns this around: the server regularly tells an outside service "I'm alive". When
those signals stop, **the outside service** alerts you.

```
vps-alert ──(every 5 minutes: "alive")──► healthchecks.io
                                              │
                          no signal for 10 minutes
                                              ▼
                                   📱 Telegram / email: "server is down"
```

vps-alert sends an HTTP request to `HEARTBEAT_URL` every `HEARTBEAT_INTERVAL` (default 5 minutes).
Since the service itself sends it, this also catches the vps-alert service being stopped or broken.
Nothing about the server is sent, just an empty request.

## Setting it up with healthchecks.io (free)

1. Sign up at [healthchecks.io](https://healthchecks.io). The free plan includes 20 checks.
2. **Add Check**, then set **Period** to 5 minutes and **Grace** to 5 minutes.
3. Copy the check's URL, like `https://hc-ping.com/xxxxxxxx-...`.
4. Under **Integrations**, connect **Telegram** (or email) so you get notified.
5. On the server:

   ```sh
   sudo nano /etc/vps-alert/vps-alert.env
   #   HEARTBEAT_URL=https://hc-ping.com/xxxxxxxx-...
   sudo systemctl restart vps-alert
   ```

Within a few minutes the check turns green (**up**).

## Uptime Kuma

If you run [Uptime Kuma](https://github.com/louislam/uptime-kuma), create a monitor of type **Push**
and use its URL as `HEARTBEAT_URL`. Run Uptime Kuma on a **different** server from the one it watches.

## Notes

- The heartbeat is off while `HEARTBEAT_URL` is empty.
- If you set `HEARTBEAT_INTERVAL`, use the same period in the monitoring service. It must be at least
  `30s`.
- Stopping the service on purpose triggers a "down" alert too. **Pause** the check before
  maintenance.
