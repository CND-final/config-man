# config-man frontend

Apple-inspired admin UI for phase 1. The frontend calls relative API paths
such as `/api/v1/auth/login`; Vite proxies `/api` to NestJS on port `3000`.

## Recommended Run

Start the NestJS backend first:

```bash
cd ../backend
npm run dev
```

Then start the Vite frontend:

```bash
npm install
npm run dev
```

Open `http://localhost:5173`.

For remote access, open `http://<remote-host>:5173`.

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
