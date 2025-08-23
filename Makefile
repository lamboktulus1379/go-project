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
	@set -a ; \
	[ -f config.env ] && . ./config.env ; \
	set +a ; \
	TARGET_PORT=$${APP_PORT:-10001} ; \
	echo "Checking port availability for $$TARGET_PORT..." ; \
	INUSE_PIDS=$$(lsof -ti tcp:$$TARGET_PORT 2>/dev/null || true) ; \
	if [ -n "$$INUSE_PIDS" ]; then \
	  echo "⚠️  Port $$TARGET_PORT is already in use by PID(s): $$INUSE_PIDS" ; \
	  while true; do \
	    echo "" ; \
	    read -p "Choose action: [k]ill existing process / [n]ew port / [a]bort: " choice ; \
	    case "$$choice" in \
	      k|K|kill) \
	        echo "🔄 Attempting to kill process(es) $$INUSE_PIDS..." ; \
	        kill $$INUSE_PIDS 2>/dev/null || true ; \
	        sleep 2 ; \
	        REMAIN=$$(lsof -ti tcp:$$TARGET_PORT 2>/dev/null || true) ; \
	        if [ -n "$$REMAIN" ]; then \
	          echo "🔥 Force killing remaining process(es) $$REMAIN..." ; \
	          kill -9 $$REMAIN 2>/dev/null || true ; \
	          sleep 1 ; \
	        fi ; \
	        FINAL_CHECK=$$(lsof -ti tcp:$$TARGET_PORT 2>/dev/null || true) ; \
	        if [ -z "$$FINAL_CHECK" ]; then \
	          echo "✅ Port $$TARGET_PORT is now free" ; \
	          APP_PORT=$$TARGET_PORT ; export APP_PORT ; \
	          break ; \
	        else \
	          echo "❌ Failed to free port $$TARGET_PORT" ; \
	        fi ; \
	        ;; \
	      n|N|new) \
	        read -p "Enter new port number: " newp ; \
	        if [[ "$$newp" =~ ^[0-9]+$$ ]] && [ "$$newp" -ge 1024 ] && [ "$$newp" -le 65535 ]; then \
	          if lsof -ti tcp:$$newp >/dev/null 2>&1; then \
	            echo "❌ Port $$newp is also in use. Try another." ; \
	          else \
	            echo "✅ Port $$newp is available" ; \
	            APP_PORT=$$newp ; export APP_PORT ; \
	            echo "Using APP_PORT=$$APP_PORT" ; \
	            break ; \
	          fi ; \
	        else \
	          echo "❌ Invalid port. Please enter a number between 1024-65535." ; \
	        fi ; \
	        ;; \
	      a|A|abort) \
	        echo "🚫 Aborting startup." ; \
	        exit 1 ; \
	        ;; \
	      *) \
	        echo "❌ Invalid choice. Please enter 'k', 'n', or 'a'." ; \
	        ;; \
	    esac ; \
	  done ; \
	else \
	  echo "✅ Port $$TARGET_PORT is available" ; \
	  APP_PORT=$$TARGET_PORT ; export APP_PORT ; \
	fi ; \
	echo "" ; \
	echo "🔐 Setting up TLS certificates..." ; \
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
	echo "" ; \
	echo "🚀 Starting HTTPS server on port $$APP_PORT with TLS using $$CERT_FILE" ; \
	echo "   📺 YouTube redirect : $$YOUTUBE_REDIRECT_URL" ; \
	echo "   📘 Facebook redirect: $$FACEBOOK_REDIRECT_URL" ; \
	echo "   🌐 Access via: https://localhost:$$APP_PORT" ; \
	echo "" ; \
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
