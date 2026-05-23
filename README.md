# config-man

Phase 1 MVP for multi-project configuration management.

## Project Structure

```text
config-man/
├── backend/   Go API using cmd/internal/pkg layout
├── frontend/  Apple-inspired Vite admin UI
└── architecture_design.md
```

## Running

This project supports two run modes. Pick one — **they can't run at the same time** (postgres port conflict).

| Situation                                          | Mode |
|----------------------------------------------------|------|
| Editing code, want hot reload                      | A    |
| Demo to teammates / professor                      | B    |
| Testing production-like setup                      | B    |
| No Go/Node installed, just want to see it run      | B    |

### Mode A — Run locally (hot reload, recommended for coding)

Start the Go backend on `3000`:

```bash
cd backend
make dev
```

Start the frontend on `5173`:

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173` locally, or `http://<remote-host>:5173` remotely.

The frontend calls relative paths such as `/api/v1/auth/login`; Vite proxies `/api` to `http://127.0.0.1:3000`.

### Mode B — Run with Docker (production-like)

No Go or Node installation required — Docker is the only prerequisite.

```bash
# First time only: create the environment file
cp .env.example .env

# From the project root — builds images and starts all three services
make docker-up
```

Open `http://localhost`. All three services (PostgreSQL, backend, frontend) start automatically.

To stop:

```bash
make docker-down
```

See [docs/DOCKER.md](docs/DOCKER.md) for environment variables, log commands, cleanup, and HTTPS preparation.

## Backend Test

```bash
cd backend
make test
```

## Demo Login

All demo accounts use password `password`.

| Email | Role |
|-------|------|
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `group-admin@config-man.local` | Group Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |

Each project belongs to exactly one group through `groupId`. Project access is enforced separately through project memberships with project-level roles: `project_admin`, `developer`, `reviewer`, and `viewer`. `system_admin` keeps global access.

## Config Import

From the Config Editor, choose a file and import it into the selected environment.

Supported formats:

- `.json`
- `.yaml` / `.yml`
- `.properties`

## Notes

- The Go backend requires `DATABASE_URL` at startup and persists data to PostgreSQL. Use `make dev` from `backend/` to load `.env` automatically.
- The API keeps the existing `/api/v1` contract used by the frontend.
- `cicd-lab/` is a separate assignment project and is not used by config-man.
