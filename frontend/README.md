# config-man frontend

Apple-inspired admin UI for phase 1. The frontend calls relative API paths
such as `/api/v1/auth/login`; Vite proxies `/api` to the Go backend on port `3000`; in Docker mode, nginx handles the same proxy.

## Running

This project supports two run modes — pick one. **They can't run at the
same time** (port conflicts on postgres).

### Mode A — Local development (hot reload, recommended for coding)

Start the Go backend first:

```bash
cd ../backend
make dev
```

Then start the Vite frontend:

```bash
npm install
npm run dev
```

Open `http://localhost:5173`.

For remote access, open `http://<remote-host>:5173`.

### Mode B — Full Docker (production-like, recommended for demo)

Everything containerized: frontend served by nginx, backend as a binary,
postgres as a service. From the project root:

```bash
docker compose up -d
```

Open `http://localhost` (port 80, no `:5173`).

See [`../docs/DOCKER.md`](../docs/DOCKER.md) for full Docker documentation.

## Proxy

`vite.config.js` forwards browser requests from `/api/*` to:

```text
http://127.0.0.1:3000
```

This avoids hardcoding the remote host in frontend code.

## Screens

- Dashboard
- Projects
- Templates
- Config Editor
- Diff & Validation
- Change Requests

## Login

Use one of the backend demo users. Every account uses password `password`.

Example: `admin@config-man.local` / `password`.

## Config Import

The Config Editor can import `.json`, `.yaml`, `.yml`, and `.properties` files
into the currently selected environment.
