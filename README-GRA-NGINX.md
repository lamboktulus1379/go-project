# Local HTTPS for gra.tulus.tech (go-project)

## Prereqs
- Homebrew Nginx (nginx service)
- mkcert installed and trust store set up
- Add to /etc/hosts:
  - `127.0.0.1 gra.tulus.tech`

## Certs
Generate local certs for gra.tulus.tech inside the repo (ignored by git):

```zsh
mkdir -p ./certs
mkcert -key-file ./certs/gra.tulus.tech-key.pem -cert-file ./certs/gra.tulus.tech-cert.pem gra.tulus.tech localhost 127.0.0.1 ::1
```

## Install Nginx site

```zsh
# copy site file
sudo cp ./local-dev/nginx/gra.tulus.tech.conf /opt/homebrew/etc/nginx/servers/gra.tulus.tech.conf

# check and reload nginx
sudo nginx -t -c /opt/homebrew/etc/nginx/nginx.conf
sudo brew services restart nginx
```

## Run the Go app
- Default port: 10001 (from `config.json`).
- TLS in the app config is enabled but Nginx already terminates TLS; you can set `app.tlsEnabled` to false for local reverse proxy, or keep as-is (Nginx to HTTPS → upstream HTTP is fine if app runs HTTP).

Run dev:
```zsh
# optional: force port
APP_PORT=10001 go run ./...
```

## Test
```zsh
curl -kI https://gra.tulus.tech
```

If you want the app reachable at HTTPS directly (no Nginx), set `app.tlsEnabled` true and ensure cert paths are valid; Nginx can still proxy to `https://127.0.0.1:10001` by adjusting `proxy_pass` accordingly.
