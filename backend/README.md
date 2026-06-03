# config-man backend

Phase 1 Go backend for the multi-project configuration management MVP.

## Stack

- Gin HTTP server
- PostgreSQL-backed store using `DATABASE_URL`
- `cmd/`, `internal/`, and `pkg/config` layout inspired by free5GC NF projects

## Structure

```text
backend/
├── cmd/main.go
├── model/
│   ├── error.go
│   ├── request.go
│   ├── project.go
│   ├── config.go
│   ├── review.go
│   ├── template.go
│   ├── user.go
│   └── validation.go
├── internal/
│   ├── context/
│   │   ├── context.go
│   │   └── request_context.go
│   ├── processor/
│   │   ├── processor.go
│   │   ├── service.go
│   │   ├── project.go
│   │   ├── config.go
│   │   ├── review_request.go
│   │   ├── validation.go
│   │   └── audit.go
│   ├── store/
│   │   ├── store.go
│   │   └── db_store.go
│   ├── response/
│   │   ├── response.go
│   │   └── error.go
│   ├── logger/
│   │   └── logger.go
│   ├── api_*.go
│   ├── middleware.go
│   ├── routes.go
│   └── server.go
└── pkg/
    ├── config/
    └── util/
```

## Local Development

```bash
cd backend
make dev
```

API base path: `http://localhost:3000/api/v1`

Environment variables:

- `DATABASE_URL`, required. Example: `postgres://config_man:config_man@localhost:5432/config_man?sslmode=disable`
- `CONFIG_MAN_HOST`, default `0.0.0.0`
- `CONFIG_MAN_PORT`, default `3000`

`make run` and `make dev` automatically create `.env` from `.env.example` when needed and export the variables for `go run ./cmd`.

## Logging

Processor actions use direct category loggers such as `logger.Project`, `logger.Config`, `logger.Auth`, `logger.Review`, and `logger.Validation`, following the category logger pattern from the referenced logger design. Logs include operation name, actor, role, project ID, environment, config key, config ID, counts, and error kind where relevant. Config values are intentionally not logged.

## Test

```bash
cd backend
make test
```

The tests cover login, project registration, masked sensitive configs, prod write permission, JSON config import, review request approval, and validation.

## Demo Users

All demo users use password `password`.

| Email | Role |
|-------|------|
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `group-admin@config-man.local` | Group Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |

## Phase 1 API

- `GET /api/v1/health`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `GET /api/v1/templates`
- `GET /api/v1/templates/base`
- `POST /api/v1/templates`
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/:projectId`
- `GET /api/v1/projects/:projectId/members`
- `PUT /api/v1/projects/:projectId/members`
- `GET /api/v1/projects/:projectId/configs` lists config documents.
- `GET /api/v1/projects/:projectId/configs?env=dev` lists config documents with their environment entries.
- `POST /api/v1/projects/:projectId/configs` creates a config document, or creates an entry for legacy clients when the body has no `name`.
- `POST /api/v1/projects/:projectId/configs/import`
- `PUT /api/v1/projects/:projectId/configs/:configId` updates a config entry for legacy clients.
- `DELETE /api/v1/projects/:projectId/configs/:configId` deletes a config entry for legacy clients.
- `GET /api/v1/review-requests`
- `GET /api/v1/projects/:projectId/review-requests`
- `POST /api/v1/review-requests`
- `PUT /api/v1/review-requests/:requestId/approve`
- `PUT /api/v1/review-requests/:requestId/reject`


## Infrastructure Templates

`GET /api/v1/templates` returns the shared infrastructure catalog plus the authenticated user's personal templates. The shared catalog currently includes `global-logging-template` and `spring-boot-base-template`. `global-logging-template` is the enterprise logging baseline with `${LOG_LEVEL_ROOT}`, `${LOG_LEVEL_COMPANY}`, and `${APP_NAME}` variables. `spring-boot-base-template` is a YAML skeleton with variables such as `${APP_PORT}`, `${DB_HOST}`, `${DB_NAME}`, `${DB_USER}`, `${DB_SECRET}`, and `${REDIS_HOST}`. `POST /api/v1/templates` creates a personal template in the DB under the current user ID, so other users cannot see or reference it. Project creation accepts an optional `templateId` and validates that the template is either shared or owned by the actor. The frontend applies templates from the New Project flow, renders variables as a form, lets the user choose yaml/json/properties output, and then extracts the rendered config through the normal config import preview flow.

## Config Import

`POST /api/v1/projects/:projectId/configs/import` supports:

- `json`: nested objects are flattened into dot keys.
- `yaml`: simple nested `key: value` YAML is flattened into dot keys.
- `properties`: `key=value` and `key:value` lines are accepted.

Sensitive-looking keys such as `password`, `secret`, `token`, `credential`, and `database.url` are marked sensitive automatically.
