# config-man backend

Phase 1 backend for a multi-project configuration management system.

## Stack

- NestJS
- PostgreSQL
- Prisma

## Local Development

From the repository root:

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

## API Smoke Test

Keep the backend running, then open another terminal:

```bash
cd backend
npm run test:api
```

The smoke test checks health, login, current user, projects, config entries, and review requests through real HTTP calls.

Use the `x-actor` header to simulate the current user, for example:

```bash
curl -H "x-actor: alice" http://localhost:3000/api/v1/projects
```

## Phase 1 API

- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/:projectId`
- `GET /api/v1/projects/:projectId/configs?env=dev`
- `POST /api/v1/projects/:projectId/configs`
- `POST /api/v1/projects/:projectId/configs/import`
- `PUT /api/v1/projects/:projectId/configs/:configId`
- `DELETE /api/v1/projects/:projectId/configs/:configId`
- `POST /api/v1/projects/:projectId/validate`
- `GET /api/v1/review-requests`
- `POST /api/v1/review-requests`
- `PUT /api/v1/review-requests/:requestId/approve`
- `PUT /api/v1/review-requests/:requestId/reject`

## Demo Users

All demo users use password `password`.

| Email | Role |
|-------|------|
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |
