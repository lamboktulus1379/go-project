## Makefile for Go Project (Liquibase + Build Helpers)

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
	echo "  tidy                 Go mod tidy" ; \
	echo "  test                 Run Go tests" ; \
	echo "Environment overrides: LB=<path to liquibase> COUNT=<n>" ;

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

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test ./... -count=1
