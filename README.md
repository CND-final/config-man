# config-man

Phase 1 MVP for multi-project configuration management.

## Project Structure

```text
config-man/
├── backend/   Go API using cmd/internal/pkg layout
├── frontend/  Apple-inspired Vite admin UI
└── architecture_design.md
```

## Recommended Run

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

- The Go backend requires `DATABASE_URL` at startup and persists data to PostgreSQL. Use `make dev` from `backend/` to load `.env` automatically.
- The API keeps the existing `/api/v1` contract used by the frontend.
- `cicd-lab/` is a separate assignment project and is not used by config-man.
