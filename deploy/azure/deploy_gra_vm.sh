#!/usr/bin/env bash
set -euo pipefail

# Deploy go-project to Azure VM behind Nginx + Let's Encrypt as gra.tulus.space

# Config (override via environment variables)
DOMAIN="${DOMAIN:-gra.tulus.space}"
APP_PORT="${APP_PORT:-10011}"
SERVICE_NAME="${SERVICE_NAME:-gra}"

# VM connection (override these for your environment)
VM_HOST="${VM_HOST:-57.158.26.137}"
VM_USER="${VM_USER:-gra}"
VM_SSH_KEY="${VM_SSH_KEY:-$HOME/Projects/Typing/deploy/infra-azure/gra-ssh.pem}"

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/dist"
RELEASE_TGZ="$BUILD_DIR/${SERVICE_NAME}-release.tgz"

echo "==> Building Linux AMD64 binary"
mkdir -p "$BUILD_DIR"
pushd "$PROJECT_ROOT" >/dev/null
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/${SERVICE_NAME}-server"
popd >/dev/null

echo "==> Preparing release archive"
cat >"$BUILD_DIR/gra.service" <<EOF
[Unit]
Description=Go Project (${SERVICE_NAME})
After=network.target

[Service]
User=${VM_USER}
Group=${VM_USER}
Environment=ENV=prod
Environment=APP_PORT=${APP_PORT}
Environment=TLS_ENABLED=0
WorkingDirectory=/opt/${SERVICE_NAME}
ExecStart=/opt/${SERVICE_NAME}/${SERVICE_NAME}-server
Restart=always
RestartSec=3
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

cat >"$BUILD_DIR/nginx-${DOMAIN}.conf" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};
    location /.well-known/acme-challenge/ { root /var/www/certbot; }
    location / { return 301 https://\$host\$request_uri; }
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN};

    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }

    location /healthz {
        proxy_pass http://127.0.0.1:${APP_PORT}/healthz;
        proxy_set_header Host \$host;
    }
}
EOF

# Optional production config (can be edited after first deploy)
cat >"$BUILD_DIR/config-prod.json" <<'EOF'
{
  "app": {
    "port": 10011,
    "secretKey": "change_me",
    "tlsEnabled": false
  },
  "logger": { "format": "2006-02-01" },
  "youtube": {
    "apiKey": "",
    "clientId": "",
    "clientSecret": "",
    "redirectURI": "https://gra.tulus.space/auth/youtube/callback",
    "channelId": "",
    "scopes": [
      "https://www.googleapis.com/auth/youtube.readonly",
      "https://www.googleapis.com/auth/youtube.upload",
      "https://www.googleapis.com/auth/youtube.force-ssl"
    ]
  },
  "oauth": {
    "facebook": {
      "clientId": "",
      "clientSecret": "",
      "redirectURI": "https://gra.tulus.space/auth/facebook/callback"
    },
    "twitter": {"clientId": "", "clientSecret": "", "redirectURI": ""}
  }
}
EOF

tar czf "$RELEASE_TGZ" -C "$BUILD_DIR" "${SERVICE_NAME}-server" "gra.service" "nginx-${DOMAIN}.conf" config-prod.json

echo "==> Uploading release to VM ${VM_USER}@${VM_HOST}"
ssh_opts=(-i "$VM_SSH_KEY" -o StrictHostKeyChecking=no)
scp "${ssh_opts[@]}" "$RELEASE_TGZ" "${VM_USER}@${VM_HOST}:/tmp/${SERVICE_NAME}-release.tgz"

echo "==> Running remote install"
ssh "${ssh_opts[@]}" "${VM_USER}@${VM_HOST}" bash -lc "'
set -euo pipefail
SERVICE_NAME="${SERVICE_NAME}"
DOMAIN="${DOMAIN}"
APP_PORT="${APP_PORT}"

sudo mkdir -p /opt/"$SERVICE_NAME"
sudo tar xzf /tmp/"$SERVICE_NAME"-release.tgz -C /opt/"$SERVICE_NAME" --no-same-owner
sudo mv /opt/"$SERVICE_NAME"/gra.service /etc/systemd/system/"$SERVICE_NAME".service
# config file already extracted to /opt/$SERVICE_NAME/config-prod.json

# Systemd
sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE_NAME" || true
sudo systemctl restart "$SERVICE_NAME"

# Nginx + Certbot
if ! command -v nginx >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y nginx; fi
if ! command -v certbot >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y certbot python3-certbot-nginx; fi
sudo mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled /var/www/certbot

# Phase 1: minimal HTTP config for ACME challenge (template + sed)
cat > /tmp/ng-http.tmpl <<'EOHTTP'
server {
  listen 80;
  server_name __DOMAIN__;
  location /.well-known/acme-challenge/ { root /var/www/certbot; }
  # Optional: proxy HTTP to app so service is reachable pre-cert
  location / {
    proxy_pass http://127.0.0.1:__APP_PORT__;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto http;
  }
}
EOHTTP
sed -e "s/__DOMAIN__/$DOMAIN/g" -e "s/__APP_PORT__/$APP_PORT/g" /tmp/ng-http.tmpl | sudo tee /etc/nginx/sites-available/"$DOMAIN" >/dev/null
sudo ln -sf /etc/nginx/sites-available/"$DOMAIN" /etc/nginx/sites-enabled/"$DOMAIN"
sudo nginx -t
sudo systemctl reload nginx || sudo systemctl restart nginx

# Obtain/renew cert
sudo certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "${CERTBOT_EMAIL:-lamboktulus1379@gmail.com}" --redirect || true

# Phase 2: apply final TLS reverse proxy config
if [ -f "/opt/$SERVICE_NAME/nginx-$DOMAIN.conf" ]; then
  sudo mv "/opt/$SERVICE_NAME/nginx-$DOMAIN.conf" "/etc/nginx/sites-available/$DOMAIN"
  sudo ln -sf "/etc/nginx/sites-available/$DOMAIN" "/etc/nginx/sites-enabled/$DOMAIN"
  sudo nginx -t && sudo systemctl reload nginx || sudo systemctl restart nginx || true
fi

echo "==> Health checks"
sleep 2
curl -fsS http://127.0.0.1:"$APP_PORT"/ >/dev/null && echo "Local app OK"
curl -fsS https://"$DOMAIN"/ -m 10 -k >/dev/null && echo "Public HTTPS OK"
'
"

echo "==> Done. Visit: https://${DOMAIN}"

echo "Tips:\n- To override VM details: VM_HOST=... VM_USER=... VM_SSH_KEY=... DOMAIN=gra.tulus.space APP_PORT=${APP_PORT} bash deploy/azure/deploy_gra_vm.sh"
