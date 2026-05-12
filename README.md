# config-man

Phase 1 MVP for multi-project configuration management.

## Project Structure

```text
config-man/
├── backend/   NestJS + Prisma + PostgreSQL API
├── frontend/  Apple-inspired static admin prototype
└── architecture_design.md
```

## Frontend Prototype

The first frontend is a no-build static prototype with mock data.

```bash
cd frontend
python3 -m http.server 5173
```

Open `http://localhost:5173`.

You can also open `frontend/index.html` directly in a browser.

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

## Notes

- `frontend/` currently uses mock data for demo clarity.
- `backend/` already exposes the phase 1 project and config APIs.
- `cicd-lab/` is a separate assignment project and is not used by config-man.
