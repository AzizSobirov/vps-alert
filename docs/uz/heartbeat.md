# Heartbeat

O'chib qolgan server o'chganini o'zi xabar qila olmaydi. Server elektrsiz qolsa, qotib qolsa yoki
hosting uni to'xtatsa, vps-alert ham u bilan birga to'xtaydi va chat shunchaki jim bo'lib qoladi.
Jimjitlik esa "hammasi yaxshi"da ham, "server o'lik"da ham bir xil ko'rinadi.

Heartbeat buni teskari qiladi: server muntazam ravishda tashqi xizmatga "men tirikman" deb xabar
beradi. Bu signallar to'xtasa, sizni **tashqi xizmat** ogohlantiradi.

```
vps-alert ──(har 5 daqiqada: "tirikman")──► healthchecks.io
                                               │
                           10 daqiqa signal kelmasa
                                               ▼
                                    📱 Telegram / email: "server javob bermayapti"
```

vps-alert har `HEARTBEAT_INTERVAL` da (standart 5 daqiqa) `HEARTBEAT_URL` ga HTTP so'rov yuboradi.
Signalni servisning o'zi yuborgani uchun bu vps-alert servisi to'xtashi yoki buzilishini ham aniqlaydi.
Server haqida hech qanday ma'lumot yuborilmaydi, faqat bo'sh so'rov boradi.

## healthchecks.io bilan sozlash (bepul)

1. [healthchecks.io](https://healthchecks.io) da ro'yxatdan o'ting. Bepul tarifda 20 ta tekshiruv bor.
2. **Add Check** bosing va **Period** ni 5 daqiqa, **Grace** ni 5 daqiqa qiling.
3. Check URL'ini nusxalang, u `https://hc-ping.com/xxxxxxxx-...` ko'rinishida bo'ladi.
4. **Integrations** bo'limida **Telegram** (yoki email) ni ulang, shunda xabar olasiz.
5. Serverda:

   ```sh
   sudo nano /etc/vps-alert/vps-alert.env
   #   HEARTBEAT_URL=https://hc-ping.com/xxxxxxxx-...
   sudo systemctl restart vps-alert
   ```

Bir necha daqiqada check yashil (**up**) bo'ladi.

## Uptime Kuma

[Uptime Kuma](https://github.com/louislam/uptime-kuma) ishlatsangiz, **Push** turidagi monitor yarating
va uning URL'ini `HEARTBEAT_URL` ga yozing. Uptime Kuma o'zi kuzatayotgan serverdan **boshqa** serverda
turishi kerak.

## Eslatmalar

- `HEARTBEAT_URL` bo'sh bo'lsa, heartbeat o'chiq turadi.
- `HEARTBEAT_INTERVAL` ni o'zgartirsangiz, monitoring xizmatida ham o'sha davrni qo'ying. U kamida
  `30s` bo'lishi kerak.
- Servisni ataylab to'xtatsangiz ham "down" alerti keladi. Texnik ishlar oldidan check'ni **Pause**
  qiling.
