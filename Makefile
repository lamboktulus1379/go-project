## Makefile for Go Project (Liquibase + Build Helpers)

SHELL := /bin/bash
.ONESHELL:

LB ?= liquibase
PG_DIR := liquibase/my_project_postgres/sql
MYSQL_DIR := liquibase/my-project/sql
COUNT ?= 1

.PHONY: help
help:
	@echo "Available targets:" ; \
	echo "  help                 Show this help" ; \
	echo "  pg-update            Apply pending PostgreSQL changesets" ; \
	echo "  pg-update-sql        Preview PostgreSQL SQL (no apply)" ; \
	echo "  pg-clear             Clear Liquibase checksums (PostgreSQL)" ; \
	echo "  pg-rollback          Roll back COUNT (default: 1) PostgreSQL changesets (use COUNT=n)" ; \
	echo "  mysql-update         Apply pending MySQL changesets" ; \
	echo "  mysql-update-sql     Preview MySQL SQL (no apply)" ; \
	echo "  mysql-clear          Clear Liquibase checksums (MySQL)" ; \
	echo "  mysql-rollback       Roll back COUNT MySQL changesets" ; \
	echo "  build                Compile Go server" ; \
	echo "  run                  Run server (dev)" ; \
	@echo "  run-https            Run server over HTTPS with local self-signed cert" ; \
	echo "  run-https-10010     Run server over HTTPS on port 10010" ; \
	echo "  run-http-10001      Run server over HTTP on port 10001" ; \
	echo "  tidy                 Go mod tidy" ; \
	echo "  test                 Run Go tests" ; \
	echo "Environment overrides: LB=<path to liquibase> COUNT=<n>" ;
	@echo "  deploy               Build and deploy to remote via deploy/deploy.sh" ; \
	echo "    Variables: SSH_HOST, SSH_USER, SSH_PORT, DOMAIN, CERTBOT=0|1, CERTBOT_EMAIL" ;
	echo "  deploy-mac           Local HTTPS via Homebrew Nginx + mkcert (domain maps to 127.0.0.1)" ; \
	echo "    Variables: DOMAIN (default: gra.tulus.tech), APP_PORT (default: 10010)" ;

## -----------------
## PostgreSQL Targets
## -----------------
.PHONY: pg-update
pg-update:
	cd $(PG_DIR) && $(LB) update

.PHONY: pg-update-sql
pg-update-sql:
	cd $(PG_DIR) && $(LB) updateSQL

.PHONY: pg-clear
pg-clear:
	cd $(PG_DIR) && $(LB) clearChecksums

.PHONY: pg-rollback
pg-rollback:
	cd $(PG_DIR) && $(LB) rollbackCount $(COUNT)

## -------------
## MySQL Targets
## -------------
.PHONY: mysql-update
mysql-update:
	cd $(MYSQL_DIR) && $(LB) update

.PHONY: mysql-update-sql
mysql-update-sql:
	cd $(MYSQL_DIR) && $(LB) updateSQL

.PHONY: mysql-clear
mysql-clear:
	cd $(MYSQL_DIR) && $(LB) clearChecksums

.PHONY: mysql-rollback
mysql-rollback:
	cd $(MYSQL_DIR) && $(LB) rollbackCount $(COUNT)

## -------------
## Go Utilities
## -------------

.PHONY: build
build:
	go build -o bin/server ./

.PHONY: run
run:
	go run main.go

.PHONY: run-https
run-https:
	@mkdir -p certs
	set -a ; \
	[ -f config.env ] && . ./config.env ; \
	set +a ; \
	if [ -f certs/localhost.crt ] && [ -f certs/localhost.key ]; then \
		CERT_FILE=$$(pwd)/certs/localhost.crt ; \
		KEY_FILE=$$(pwd)/certs/localhost.key ; \
		echo "Using provided certs: $$CERT_FILE $$KEY_FILE" ; \
	elif command -v mkcert >/dev/null 2>&1; then \
		echo "Using mkcert to generate a trusted localhost certificate..." ; \
		mkcert -install >/dev/null 2>&1 || true ; \
		if [ ! -f certs/localhost+2.pem ] || [ ! -f certs/localhost+2-key.pem ]; then \
			( cd certs && mkcert localhost 127.0.0.1 ::1 ); \
		fi ; \
		CERT_FILE=$$(pwd)/certs/localhost+2.pem ; \
		KEY_FILE=$$(pwd)/certs/localhost+2-key.pem ; \
	else \
		echo "mkcert not found; generating self-signed cert (may show browser warning)..." ; \
		if [ ! -f certs/dev.localhost.crt ] || [ ! -f certs/dev.localhost.key ]; then \
			openssl req -x509 -newkey rsa:2048 -nodes -keyout certs/dev.localhost.key -out certs/dev.localhost.crt -days 365 -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1" ; \
		fi ; \
		CERT_FILE=$$(pwd)/certs/dev.localhost.crt ; \
		KEY_FILE=$$(pwd)/certs/dev.localhost.key ; \
	fi ; \
	echo " Access Token: $$YOUTUBE_ACCESS_TOKEN" ; \
	echo " Refresh Token: $$YOUTUBE_REFRESH_TOKEN" ; \
	echo "Starting server with TLS using $$CERT_FILE" ; \
	echo "  YouTube redirect : $$YOUTUBE_REDIRECT_URL" ; \
	echo "  Facebook redirect: $$FACEBOOK_REDIRECT_URL" ; \
	[ -n "$(APP_PORT)" ] && export APP_PORT=$(APP_PORT) ; \
	export TLS_ENABLED=1 TLS_CERT_FILE=$$CERT_FILE TLS_KEY_FILE=$$KEY_FILE ; \
	export YOUTUBE_REDIRECT_URL FACEBOOK_REDIRECT_URL ; \
	go run main.go

.PHONY: run-https-10010
run-https-10010:
	@$(MAKE) run-https APP_PORT=10010

.PHONY: run-http-10001
run-http-10001:
	@set -a ; \
	[ -f config.env ] && . ./config.env ; \
	set +a ; \
	echo "Starting server over HTTP on port 10001" ; \
	export TLS_ENABLED=0 APP_PORT=10001 ; \
	go run main.go

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test ./... -count=1

.PHONY: deploy
deploy:
	@bash deploy/deploy.sh

.PHONY: deploy-mac
deploy-mac:
	@bash deploy/mac/deploy-local-mac.sh
