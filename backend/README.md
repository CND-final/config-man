# config-man backend

Phase 1 Go backend for the multi-project configuration management MVP.

## Stack

- Gin HTTP server
- In-memory seeded store for the first rewrite
- `cmd/`, `internal/`, and `pkg/config` layout inspired by free5GC NF projects

This directory is now Go-only; the previous NestJS/Prisma backend has been removed.

## Structure

```text
backend/
├── cmd/main.go
├── model/
│   ├── request.go
│   ├── user.go
│   ├── project.go
│   ├── config.go
│   ├── review.go
│   ├── template.go
│   └── validation.go
├── internal/
│   ├── processor/
│   │   ├── service.go
│   │   ├── store.go
│   │   ├── permissions.go
│   │   ├── config_parser.go
│   │   └── validation.go
│   ├── logger/
│   │   └── logger.go
│   ├── api_auth.go
│   ├── api_configs.go
│   ├── api_health.go
│   ├── api_projects.go
│   ├── api_reviews.go
│   ├── api_templates.go
│   ├── api_validation.go
│   ├── response.go
│   ├── routes.go
│   └── server.go
└── pkg/config/
    ├── config.go
    └── factory.go
```

## Local Development

```bash
cd backend
go run ./cmd
```

API base path: `http://localhost:3000/api/v1`

Environment variables:

- `CONFIG_MAN_HOST`, default `0.0.0.0`
- `CONFIG_MAN_PORT`, default `3000`

## Test

```bash
cd backend
go test ./...
```

The tests cover login, project registration, masked sensitive configs, prod write permission, JSON config import, review request approval, and validation.

## Demo Users

All demo users use password `password`.

| Email | Role |
|-------|------|
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |

## Phase 1 API

- `GET /api/v1/health`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `GET /api/v1/templates/base`
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
- `GET /api/v1/projects/:projectId/review-requests`
- `POST /api/v1/review-requests`
- `PUT /api/v1/review-requests/:requestId/approve`
- `PUT /api/v1/review-requests/:requestId/reject`

## Config Import

`POST /api/v1/projects/:projectId/configs/import` supports:

- `json`: nested objects are flattened into dot keys.
- `yaml`: simple nested `key: value` YAML is flattened into dot keys.
- `properties`: `key=value` and `key:value` lines are accepted.

Sensitive-looking keys such as `password`, `secret`, `token`, `credential`, and `database.url` are marked sensitive automatically.
