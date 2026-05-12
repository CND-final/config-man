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

Use the `x-actor` header to simulate the current user, for example:

```bash
curl -H "x-actor: alice" http://localhost:3000/api/v1/projects
```

## Phase 1 API

- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/:projectId`
- `GET /api/v1/projects/:projectId/configs?env=dev`
- `POST /api/v1/projects/:projectId/configs`
- `PUT /api/v1/projects/:projectId/configs/:configId`
- `DELETE /api/v1/projects/:projectId/configs/:configId`
- `POST /api/v1/projects/:projectId/validate`
