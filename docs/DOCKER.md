# Docker Deployment Guide

This document covers running config-man as a fully containerized stack using Docker Compose.

---

## Two Operating Modes

| | Local dev (`make dev`) | Docker stack (`make docker-up`) |
|---|---|---|
| **Backend** | `go run ./cmd` on host | Alpine container |
| **Frontend** | Vite dev server (HMR) | nginx serving `dist/` |
| **PostgreSQL** | Docker container | Docker container |
| **Config file** | `backend/.env` | root `.env` |
| **DATABASE_URL host** | `localhost` | `postgres` (service name) |
| **Frontend port** | 5173 | 80 |
| **Hot reload** | Yes (backend + frontend) | No (rebuild required) |
| **Started from** | `backend/` directory | project root |

Use **local dev** when actively writing code. Use **Docker stack** to verify the full deployment chain or to hand off to teammates who don't have Go/Node installed.

> Both modes start a PostgreSQL container on host port 5432. Running them simultaneously causes a port conflict. Use one at a time.

---

## Quick Start

```bash
# From the project root
make docker-up
```

Wait for all three services to become healthy (30–60 seconds on first run — images are built from source).

Open `http://localhost` in your browser.

Demo accounts (all use password `password`):

| Email | Role |
|-------|------|
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |

---

## Environment Variables

The root `.env` file is read automatically by `docker-compose.yml`.
`.env` is listed in `.gitignore`, so you must create it before the first run:

```bash
cp .env.example .env
```

The defaults in `.env.example` work out of the box for local development.
All variables also have inline defaults in `docker-compose.yml`, so the stack
starts even without a `.env` file — but having the file makes values explicit and editable.

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://config_man:config_man@postgres:5432/config_man?sslmode=disable` | Must use `postgres` (Docker service name), not `localhost` |
| `POSTGRES_DB` | `config_man` | PostgreSQL database name |
| `POSTGRES_USER` | `config_man` | PostgreSQL user |
| `POSTGRES_PASSWORD` | `config_man` | PostgreSQL password |
| `POSTGRES_PORT` | `5432` | Host port mapped to PostgreSQL |
| `CONFIG_MAN_HOST` | `0.0.0.0` | Backend listen address |
| `CONFIG_MAN_PORT` | `3000` | Backend internal port |
| `BACKEND_PORT` | `3000` | Host port mapped to backend |
| `FRONTEND_PORT` | `80` | Host port mapped to frontend (nginx) |

---

## Starting and Stopping

```bash
# Build images from source and start all services
make docker-up

# Stop all services (data volume is preserved)
make docker-down

# Stop and delete data volumes (full reset — all data is lost)
docker compose down -v

# Rebuild and restart one service after a code change
docker compose up -d --build backend
docker compose up -d --build frontend
```

---

## Viewing Logs

```bash
# Stream logs from all services
docker compose logs -f

# Stream logs from one service
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f postgres
```

---

## Cleaning Up

```bash
# Remove containers and network; keep data volume
make docker-down

# Remove containers, network, and data volume
docker compose down -v

# Remove locally built images (forces full rebuild next time)
docker compose down --rmi local

# Nuclear option: remove everything including cached layers
docker compose down -v --rmi local
docker system prune -f
```

---

## Troubleshooting

### Backend fails to connect: "connection refused" or "dial tcp"

**Cause**: `DATABASE_URL` still contains `localhost` instead of the `postgres` service name.

Inside a Docker container, `localhost` refers to the container itself — not to other containers. The correct hostname is the Docker Compose service name: `postgres`.

**Verify**:
```bash
docker compose config | grep DATABASE_URL
# Expected: postgres://...@postgres:5432/...
# Wrong:    postgres://...@localhost:5432/...
```

**Fix**: Edit root `.env` and ensure `DATABASE_URL` uses `@postgres:5432`.

---

### Port 5432 already in use

**Cause**: `make dev` left a PostgreSQL container running (from `backend/docker-compose.yml`).

**Fix**: Stop it before running the full stack.
```bash
cd backend && make db-down
# then retry:
cd .. && make docker-up
```

---

### Frontend shows "502 Bad Gateway" on API calls

**Cause**: nginx cannot reach `http://backend:3000`. The backend container is either not running or its health check is still failing.

**Diagnose**:
```bash
docker compose ps                # check all services show "healthy"
docker compose logs backend      # look for startup errors or panics
```

---

### Images are stale after code changes

`make docker-up` always passes `--build`, which rebuilds from source. If you used `docker compose up -d` directly (without `--build`), cached images are reused.

Always use `make docker-up` when you have changed Go or frontend code.

---

### `make docker-up` is very slow

The first run builds both Go and Node images from scratch — expect 2–5 minutes. Subsequent runs reuse cached layers and are much faster unless `go.mod`, `go.sum`, or `package-lock.json` changed.

---

## Future: Enabling HTTPS

When HTTPS is needed, choose one of the following approaches.

### Option 1: nginx handles SSL directly

Best for: single-VM deployments where you manage certificates yourself.

Steps:
1. Obtain certificate files (`fullchain.pem`, `privkey.pem`) — e.g., via `certbot --standalone`.
2. Place them in `./certs/` at the project root.
3. In `docker-compose.yml`, uncomment:
   ```yaml
   ports:
     - "443:443"
   volumes:
     - ./certs:/etc/nginx/certs:ro
   ```
4. In `frontend/nginx.conf`, uncomment the `listen 443 ssl` block and add an HTTP→HTTPS redirect:
   ```nginx
   server {
       listen 80;
       server_name _;
       return 301 https://$host$request_uri;
   }
   ```
   The `proxy_set_header X-Forwarded-Proto $scheme` header (already present) ensures the backend knows the original request was HTTPS.
5. Rebuild: `make docker-up`.

### Option 2: Add a reverse proxy container (Caddy or Traefik)

Best for: automated certificate management with Let's Encrypt.

Add a fourth service to `docker-compose.yml`:

- **Caddy**: Automatically obtains and renews Let's Encrypt certificates with minimal configuration. The `frontend` service stays on port 80 (internal only, not exposed to host). Caddy terminates TLS and forwards traffic to nginx.
- **Traefik**: Label-based routing; integrates well with Docker. Similar setup to Caddy.

With this approach, `frontend/nginx.conf` and `docker-compose.yml` require no HTTPS changes — TLS is handled entirely by the proxy container.

### Option 3: Terminate TLS at the cloud or infrastructure layer

Best for: cloud VM or Kubernetes deployments.

- Use a cloud load balancer (AWS ALB, GCP HTTPS LB, Azure Application Gateway) to terminate TLS.
- The Docker Compose stack runs entirely on HTTP internally.
- No changes to `nginx.conf` or `docker-compose.yml` are needed.
- The `X-Forwarded-Proto: https` header is set by the load balancer; the backend's existing `X-Forwarded-Proto` handling will work correctly.

This is the recommended approach for production cloud deployments.
