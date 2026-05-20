SHELL := /usr/bin/env bash

ifneq (,$(wildcard .env))
include .env
export
endif

DATABASE_URL ?= postgres://config_man:config_man@localhost:5432/config_man?sslmode=disable
CONFIG_MAN_HOST ?= 0.0.0.0
CONFIG_MAN_PORT ?= 3000
export DATABASE_URL CONFIG_MAN_HOST CONFIG_MAN_PORT

.PHONY: help env db-up db-down db-logs run dev test fmt tidy clean

help:
	@echo "config-man backend commands"
	@echo "  make env      Create .env from .env.example if it does not exist"
	@echo "  make db-up    Start PostgreSQL with Docker Compose"
	@echo "  make db-down  Stop PostgreSQL"
	@echo "  make run      Run backend with .env exported"
	@echo "  make dev      Start DB, then run backend"
	@echo "  make test     Run Go tests"
	@echo "  make fmt      Format Go files"
	@echo "  make tidy     Run go mod tidy"

.env:
	cp .env.example .env

env: .env

db-up:
	docker compose up -d
	@until docker compose exec -T postgres pg_isready -U config_man -d config_man; do \
		sleep 1; \
	done

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

run: env
	@if [[ -z "$${DATABASE_URL}" ]]; then \
		echo "DATABASE_URL is required. Check backend/.env"; \
		exit 1; \
	fi
	go run ./cmd

dev: db-up run

test:
	go test ./...

fmt:
	gofmt -w cmd internal model pkg

tidy:
	go mod tidy

clean:
	docker compose down
