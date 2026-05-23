# config-man root Makefile
# Full-stack Docker Compose targets.
#
# For backend-only local development (no containers for Go/frontend):
#   cd backend && make dev

.PHONY: docker-up docker-down

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
