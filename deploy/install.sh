#!/bin/sh
# Installs or updates vps-alert as a systemd service.
#
#   curl -fsSL https://github.com/AzizSobirov/vps-alert/releases/latest/download/install.sh | sudo sh
#   sudo sh install.sh ./vps-alert-linux-amd64     # use a local binary
set -eu

REPO="AzizSobirov/vps-alert"
BIN=/usr/local/bin/vps-alert
ENV_FILE=/etc/vps-alert/vps-alert.env
UNIT=/etc/systemd/system/vps-alert.service
PAM_LINE="session optional pam_exec.so quiet $BIN -env $ENV_FILE pam-hook"

die() { echo "error: $*" >&2; exit 1; }
# Read from the terminal, so this works when piped into sh.
ask() { printf '%s' "$1" > /dev/tty; read -r REPLY < /dev/tty; }

[ "$(id -u)" -eq 0 ] || die "run as root"
[ "$(uname -s)" = Linux ] || die "only Linux is supported"
command -v systemctl > /dev/null || die "systemd is required"

case "$(uname -m)" in
	x86_64) ARCH=amd64 ;;
	aarch64 | arm64) ARCH=arm64 ;;
	*) die "unsupported architecture $(uname -m)" ;;
esac

echo "==> Installing binary"
if [ $# -ge 1 ]; then
	install -m 755 "$1" "$BIN"
else
	tmp=$(mktemp -d)
	url="https://github.com/$REPO/releases/latest/download"
	curl -fsSL "$url/vps-alert-linux-$ARCH" -o "$tmp/vps-alert-linux-$ARCH"
	curl -fsSL "$url/checksums.txt" -o "$tmp/checksums.txt"
	# Refuse to install a binary that does not match the published checksum.
	(cd "$tmp" && grep " vps-alert-linux-$ARCH\$" checksums.txt | sha256sum -c -) || die "checksum mismatch"
	install -m 755 "$tmp/vps-alert-linux-$ARCH" "$BIN"
	rm -rf "$tmp"
fi

if ! command -v restic > /dev/null; then
	echo "==> Installing restic"
	if command -v apt-get > /dev/null; then
		apt-get update -qq && apt-get install -y -qq restic
	elif command -v dnf > /dev/null; then
		dnf install -y -q restic
	else
		echo "warning: install restic manually, backups need it"
	fi
fi

if [ ! -f "$ENV_FILE" ]; then
	echo "==> Configuration"
	ask "Bot token (from @BotFather): "; token=$REPLY
	ask "Allowed chat ID: "; chat=$REPLY
	ask "Allowed user IDs (comma separated, recommended): "; users=$REPLY
	[ -n "$token" ] && [ -n "$chat" ] || die "token and chat ID are required"
	restic_pw=$(head -c 32 /dev/urandom | base64 | tr -d '/+=' | head -c 40)

	install -d -m 700 /etc/vps-alert
	umask 077
	cat > "$ENV_FILE" <<CONF
BOT_TOKEN=$token
ALLOWED_CHAT_ID=$chat
ALLOWED_USER_IDS=$users
SERVER_NAME=$(hostname)
RESTIC_PASSWORD=$restic_pw
BACKUP_LOCAL_REPO=/var/backups/vps-alert
BACKUP_PRE_COMMAND=
R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
R2_BUCKET=
R2_PREFIX=vps-alert
LOGIN_IP_WHITELIST=
HEARTBEAT_URL=
CONF
	chmod 600 "$ENV_FILE"
	echo
	echo "  Backup encryption password (RESTIC_PASSWORD):"
	echo "    $restic_pw"
	echo "  Save it in a password manager. Without it backups cannot be restored."
	echo
fi

echo "==> Sending test message"
"$BIN" -env "$ENV_FILE" test || die "test message failed, check $ENV_FILE"

if [ -f /etc/pam.d/sshd ] && ! grep -qF "$BIN" /etc/pam.d/sshd; then
	ask "Enable SSH login alerts? [Y/n] "
	case "$REPLY" in
		n | N) ;;
		*) echo "$PAM_LINE" >> /etc/pam.d/sshd && echo "    login alerts enabled" ;;
	esac
fi

echo "==> Starting service"
# Same as deploy/vps-alert.service, inlined so the script works on its own.
cat > "$UNIT" <<'UNITFILE'
[Unit]
Description=vps-alert server monitoring and backups
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=/usr/local/bin/vps-alert -env /etc/vps-alert/vps-alert.env run
Restart=always
RestartSec=5
User=root
ProtectSystem=full
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNITFILE
systemctl daemon-reload
systemctl enable --now vps-alert
systemctl restart vps-alert

echo "Done. Send /help to your bot. Logs: journalctl -u vps-alert -f"
