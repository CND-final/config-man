# config-man

Phase 1 MVP for multi-project configuration management.

## Project Structure

```text
config-man/
├── backend/   NestJS + Prisma + PostgreSQL API
├── frontend/  Apple-inspired static admin prototype
└── architecture_design.md
```

## Frontend

Recommended development setup: run Vite on `5173` and let it proxy `/api` to the NestJS backend on `3000`.

```bash
cd backend
npm run dev
```

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173` locally, or `http://<remote-host>:5173` remotely.

## Backend API

```bash
cd backend
cp .env.example .env
npm install
docker compose up -d
npm run prisma:migrate -- --name init
npm run prisma:seed
npm run dev
```

API base path: `http://localhost:3000/api/v1`

## Demo Login

All demo accounts use password `password`.

| Email | Role |
|-------|------|
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |

## Config Import

From the Config Editor, choose a file and import it into the selected environment.

Supported formats:

- `.json`
- `.yaml` / `.yml`
- `.properties`

## Notes

- `frontend/` calls `http://localhost:3000/api/v1` by default.
- `backend/` exposes login, project, config, import, validation, and review request APIs.
- `cicd-lab/` is a separate assignment project and is not used by config-man.
