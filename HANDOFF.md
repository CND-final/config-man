# config-man Handoff Notes

This file summarizes the current state of the `config-man` work so another agent can continue safely.

## Project Goal

`config-man` is a phase 1 MVP for centralized configuration management across many projects. It targets project registration, environment-aware config CRUD, config import from multiple formats, validation, review requests, and simple role-based permissions.

The product direction is:

- Frontend: Apple-inspired admin UI, simple and clean, connected to real APIs.
- Backend: Go + Gin + PostgreSQL, using a free5GC-inspired folder style.
- Deployment for local development: PostgreSQL via Docker Compose, frontend via Vite proxy.

## Current Top-Level Layout

```text
config-man/
├── architecture_design.md
├── README.md
├── HANDOFF.md
├── backend/
└── frontend/
```

## Backend Summary

Backend is implemented in Go under `backend/`.

Important layout:

```text
backend/
├── cmd/main.go
├── internal/
│   ├── api_auth.go
│   ├── api_configs.go
│   ├── api_health.go
│   ├── api_projects.go
│   ├── api_reviews.go
│   ├── api_templates.go
│   ├── context/
│   ├── logger/
│   ├── middleware.go
│   ├── processor/
│   ├── response/
│   ├── routes.go
│   ├── server.go
│   └── store/
├── model/
├── pkg/
│   ├── config/
│   └── util/
├── docker-compose.yml
├── Makefile
└── README.md
```

### Backend Stack

- Gin HTTP server
- PostgreSQL via `database/sql` with `pgx` driver
- In-memory store shape plus DB-backed initialization in `internal/store`
- `DATABASE_URL` is required at startup
- `Makefile` loads `.env` automatically for common commands

### Backend Run

```bash
cd backend
make dev
```

Expected API base:

```text
http://localhost:3000/api/v1
```

PostgreSQL is expected from Docker Compose:

```bash
cd backend
docker compose up -d
```

### Backend Tests

Last verified command:

```bash
cd backend
go test ./...
```

It passed.

### Backend API Surface

Implemented phase 1 APIs:

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
- `GET /api/v1/projects/:projectId/configs?env=dev`
- `GET /api/v1/projects/:projectId/config-history?env=dev`
- `POST /api/v1/projects/:projectId/config-history/rollback`
- `POST /api/v1/projects/:projectId/configs`
- `POST /api/v1/projects/:projectId/configs/extract`
- `POST /api/v1/projects/:projectId/configs/import`
- `PUT /api/v1/projects/:projectId/configs/:configId`
- `GET /api/v1/projects/:projectId/configs/:configId/versions` (legacy per-entry history; UI now uses config revisions)
- `POST /api/v1/projects/:projectId/configs/:configId/rollback` (legacy per-entry rollback; UI now uses config revisions)
- `DELETE /api/v1/projects/:projectId/configs/:configId`
- `GET /api/v1/review-requests`
- `GET /api/v1/projects/:projectId/review-requests`
- `POST /api/v1/review-requests`
- `PUT /api/v1/review-requests/:requestId/approve`
- `PUT /api/v1/review-requests/:requestId/reject`
- `GET /api/v1/users`
- `GET /api/v1/groups`
- `POST /api/v1/groups`
- `GET /api/v1/groups/:groupId`
- `PATCH /api/v1/groups/:groupId`
- `DELETE /api/v1/groups/:groupId`
- `POST /api/v1/groups/:groupId/members`
- `DELETE /api/v1/groups/:groupId/members/:userId`

### Auth And Permissions

Current auth is simple demo-token auth:

- `POST /api/v1/auth/login` returns token equal to seeded user ID.
- Protected APIs use middleware in `internal/middleware.go`.
- Token is read from `Authorization: Bearer <token>` or `X-Actor`.
- Actor is placed into `internal/context.RequestContext`.

Demo users all use password `password`:

| Email | Role |
| --- | --- |
| `admin@config-man.local` | System Admin |
| `project-admin@config-man.local` | Project Admin |
| `group-admin@config-man.local` | Group Admin |
| `developer@config-man.local` | Developer |
| `reviewer@config-man.local` | Reviewer |
| `viewer@config-man.local` | Viewer |

Permissions live in `backend/pkg/util/permissions.go`; project-level membership is persisted in `project_members`, while processors only load the project/member data and map authorization failures to app errors.

### Processor Layer

`backend/internal/processor` is the business behavior layer. Store is only for persistence, while processor decides what actions happen.

Files:

- `processor.go`: `Processor` struct and constructor
- `service.go`: auth/login/template behavior
- `project.go`: project list/create/get
- `config.go`: config list/create/update/delete/import plus version history and rollback behavior
- `review_request.go`: review request list/create/approve/reject
- `group.go`: group list/create/update/delete plus group member add/remove
- `audit.go`: audit/version helper creation

### Store Layer

`backend/internal/store` owns data persistence operations.

Important intent:

- `store` should stay focused on saving/fetching data.
- Do not put business permission decisions in `store`.
- Processor should orchestrate calls and enforce behavior.

### Group And Group Members

Group management is partially implemented for the role architecture discussion. The current backend supports creating groups and adding/removing group members from the frontend Group modal.

Current behavior:

- `system_admin` can list users, create groups, update/delete groups, and add/remove group members.
- `group_admin` user-role accounts can list users, create groups, and manage members for groups they belong to; when they create a group they are added as that group's `group_admin` member.
- Non-system users can list only groups they belong to.
- A per-group `group_admin` membership role is exposed in the frontend member role menu.
- Groups are persisted in `groups`; membership is persisted in `group_members` with `group_id`, `user_id`, and `role`.
- Project ownership is 1-to-1 through `projects.group_id`; `Group.Projects` is loaded by querying projects with that `group_id`, not by a join table.
- Project access remains explicit through `project_members`; group ownership does not automatically grant project read/write permissions yet.
- Demo users still live in the in-memory seed user list, so `group_members.user_id` is not yet a foreign key.

Important files:

- `backend/model/group.go`
- `backend/internal/api_group.go`
- `backend/internal/processor/group.go`
- `backend/internal/store/group_stroe.go` (typo retained from existing file name)
- `backend/internal/db_store/db_group.go`
- `backend/internal/store/user_store.go`

### Response And Errors

Errors use project-wide model types in `backend/model/error.go`:

- `model.ErrorDetail`
- `model.ErrorKind`
- constructors such as `model.InvalidInput`, `model.NotFound`, `model.Forbidden`, etc.

HTTP mapping is in `backend/internal/response/error.go`.

This keeps processor from depending directly on HTTP status codes.

### Infrastructure Templates

Template catalog now includes shared infrastructure templates plus per-user custom templates. Shared templates are `global-logging-template` and `spring-boot-base-template`; `global-logging-template` is the logging baseline. `POST /templates` stores custom templates in `custom_templates` with `owner_user_id`; `GET /templates` appends only the authenticated user's custom templates, so other users cannot see or reference them. `POST /projects` accepts optional `templateId` and validates that it is either shared or owned by the actor. The frontend Templates page creates and displays templates only; it does not show Apply Template buttons on cards. Template cards keep the `shared`/`personal` visibility tag, but do not show format/type badges such as `yaml` or `base`. Template application starts from the New Project modal's Choose from Template Library button: the project form draft is preserved, the UI switches to Templates, template cards show body previews and become selectable, Cancel returns to Create Project, and Use Template returns to Create Project with the chosen variables/output format saved. Creating the project then opens Extract Config preview for the rendered template.

### Config Import

Config import has a parse-only preview endpoint and a write endpoint:

```text
POST /api/v1/projects/:projectId/configs/extract
POST /api/v1/projects/:projectId/configs/import
```

The frontend uses `extract` first, shows created/updated/unchanged counts plus parsed key/value rows, and only calls `import` after the user confirms Apply Import.

Supported formats:

- JSON
- YAML / YML
- `.properties`

Parsing helpers are in `backend/pkg/util/config_parser.go`.

Sensitive-looking keys are detected in `backend/pkg/util/sensitive.go` and are not logged as values.

### Config Version History And Rollback

Config create/update/import/delete/rollback operations now write environment-level `model.ConfigRevision` records in addition to the older per-entry `model.ConfigVersion` rows. The UI uses config revisions, because history should represent the whole config for a project/environment rather than a single key/value.

Current revision APIs:

```text
GET /api/v1/projects/:projectId/config-history?env=dev
POST /api/v1/projects/:projectId/config-history/rollback
```

Revision rollback restores the selected revision's full entry set for that environment, removes keys not present in the revision, writes a new revision for the restored state, and records an audit log. Sensitive values are masked by default in revision history unless `revealSensitive=true` is supplied by an allowed role.

Legacy per-entry APIs still exist for now:

```text
GET /api/v1/projects/:projectId/configs/:configId/versions
POST /api/v1/projects/:projectId/configs/:configId/rollback
```

### Logger State

Logger is currently simplified in `backend/internal/logger/logger.go`.

Current style:

```go
logger.Project.Info("create project requested")
logger.Config.Warn("create config denied")
logger.DB.Info("DATABASE_URL detected; opening PostgreSQL connection")
```

Important: a previous `ConfigManLogger` struct and old `MainLog/APILog/DBLog` aliases were removed. Only package-level category loggers remain:

- `logger.Main`
- `logger.API`
- `logger.Processor`
- `logger.Store`
- `logger.DB`
- `logger.Auth`
- `logger.Project`
- `logger.Config`
- `logger.Review`
- `logger.Template`

The user has been pushing toward a free5GC-like direct category logger style. Avoid reintroducing a `processor/logging.go` helper or local `log := logger.Project.With(...)` unless the user explicitly changes direction.

Current caveat: `project.go` and `config.go` may still contain `fields := []any{...}` helper slices. The user recently questioned this pattern and may prefer logs to be direct message-only or direct key-value calls. If continuing logger cleanup, remove `fields` and write direct logger calls.

## Frontend Summary

Frontend is under `frontend/` and uses Vite.

Important files:

```text
frontend/
├── index.html
├── app.js                 # module entrypoint only
├── src/
│   ├── actions.js         # user workflows and mutations
│   ├── api.js             # API wrapper and auth header handling
│   ├── data.js            # API-backed state loading
│   ├── dom.js             # DOM selectors and shell helpers
│   ├── events.js          # event binding and app bootstrap
│   ├── render.js          # pure-ish view rendering from state
│   ├── state.js           # shared state and app constants
│   └── utils.js           # formatting/status helpers
├── styles.css
├── package.json
├── package-lock.json
└── vite.config.js
```

### Frontend Run

```bash
cd frontend
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

For remote access:

```text
http://<remote-host>:5173
```

If port 5173 is already in use, Vite will choose the next available port and print it in the terminal.

### Frontend API Proxy

`frontend/vite.config.js` proxies `/api` to the Go backend:

```text
http://127.0.0.1:3000
```

Frontend code should call relative URLs such as:

```js
fetch('/api/v1/auth/login')
```

Do not hardcode `localhost:3000` in frontend fetch calls, especially for remote access.

### Frontend Features

The frontend is an Apple-inspired admin UI and includes:

- Login screen with the config-man SVG logo from `frontend/public/config-man-logo.svg`; the old API/demo-user helper text is intentionally hidden
- Dashboard
- Projects
- Project registration modal backed by `POST /api/v1/projects`, including optional visible-template application
- Templates with global logging/Spring Boot skeleton support, personal template creation, and variable extraction
- Sidebar navigation uses SVG image icons from `frontend/public/nav-*.svg` instead of letter codes
- Config Editor, entered through clickable project cards instead of sidebar navigation; the left rail shows standard config files (`application.yaml`, `redis.yaml`, `security.json`) for the selected project instead of other projects
- Config header shows the current environment revision version
- GitHub-style history icon sits in the Config page top-right action area and opens the environment-level config history modal backed by `GET /config-history?env=...`
- Rollback previous full-config revision backed by `POST /config-history/rollback`
- Change Requests
- Group modal from the user menu, backed by `/groups` and `/users`, with group creation and member add/remove for system admins
- Config import from JSON/YAML/properties via a top-right Import Config modal, then Extract Config preview, then Apply Import
- Config export via an Export Config modal that lets the user choose `json`, `yaml`, or `properties`, then checks reveal-sensitive permission before downloading unmasked values
- Sensitive value handling in the UI
- Warning/confirmation behavior for sensitive Prod database password changes
- Working global search across dashboard/project/template/request/config surfaces
- Inline config key/value editing with a hover pencil icon and no Action column
- Bottom-right Submit Review dock shown only after local config changes

### Frontend Refactor Notes

The frontend was refactored from one large `app.js` into small ES modules under `frontend/src/`. Keep the boundaries simple:

- `state.js` owns shared mutable state and constants.
- `api.js` owns the relative `/api/v1` fetch wrapper.
- `data.js` loads API payloads into state but should not render.
- `render.js` renders DOM from state and should avoid API calls.
- `actions.js` coordinates workflows such as create project, edit config, import/export config, review decisions, config revision history, and rollback.
- `events.js` binds DOM events and bootstraps the app.
- Modal open/close workflows in `actions.js` call modal render helpers directly, so keep `renderTemplateModal`, `renderExportModal`, and `renderReviewModal` imported when touching those flows.

The app still intentionally uses vanilla JavaScript and Vite. Do not add a frontend framework unless the user asks for that direction.

Config table columns are intentionally minimal: Key, Value, Updated. The frontend no longer shows `valueType`, synthetic config status, project health badges, or staging/prod diff UI. The sidebar intentionally omits Config; users enter the editor by clicking a project card, which has a small hover scale effect without adding a hover/click shadow. File selection for imports is hidden behind the Config page Import Config action, not shown directly above the config table. History rows are intentionally compact and GitHub-like: changed-by name, version, and time stay on one line. Config key/value edits are inline: hover a cell to reveal a pencil icon, edit in place, and save on Enter or blur. Inline edit should stay visually stable: the table uses fixed column widths, the input inherits the cell font, and entering/canceling edit replaces only the affected config row instead of redrawing the whole table body. Submit Review is hidden until inline changes exist, then appears as a bottom-right floating pill. Pressing it opens a review modal that lists key/value changes before creating the review request. Import Apply is treated like manual edits and contributes to the same pending review state. Export uses `revealSensitive=true`; roles that cannot reveal sensitive values cannot export masked config.

### Frontend Build

Last verified command:

```bash
cd frontend
npm run build
```

It passed.

## Important Design Decisions

- Backend was rewritten from NestJS to Go.
- Old NestJS backend should not be revived unless requested.
- The target backend style is free5GC-inspired, with `cmd`, `internal`, `model`, and `pkg` separation.
- `processor` is behavior/business logic.
- `store` is persistence.
- `model` contains shared project data structures.
- `response` handles HTTP response mapping.
- `pkg/util` contains reusable helpers such as permissions, parsing, validation, and sensitive detection.
- Logger should be direct category loggers, not injected into processor.

## Known Follow-Up Work

Recommended next steps:

1. Clean up `project.go` and `config.go` logger calls if the user wants no `fields := []any{...}` helper slices.
2. Check all processor files for consistent logger category use.
3. Add request access logging middleware only if requested; current goal was processor-level trace logs.
4. Improve DB migrations/schema management if the project moves past phase 1 demo.
5. Decide whether group ownership should grant inherited project permissions. Today `projects.group_id` models ownership, while effective project access is still enforced through `project_members`.
6. Replace demo auth with JWT + RBAC when core workflows stabilize.
7. Add more API tests around config import formats and review request requirements. The extract preview endpoint currently has basic non-persistence coverage.
8. Ensure frontend warnings for sensitive Prod DB password updates align exactly with backend permissions/review request flow.
9. Consider adding targeted frontend tests if the vanilla JS UI keeps gaining workflow logic.
10. Reintroduce environment compare/diff only if it becomes a concrete review workflow; it was removed from the MVP UI to keep the product focused.

## Useful Commands

Backend:

```bash
cd backend
make dev
make test
go test ./...
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Database:

```bash
cd backend
docker compose up -d
```

## Notes For The Next Agent

- The user prefers direct, simple code over abstraction layers that only hide one line of work.
- The user is actively shaping architecture and asks good boundary questions. Be honest when a pattern is unnecessary.
- Do not reintroduce `processor/logging.go`.
- Do not put HTTP status directly into processor errors.
- Do not log config values, especially sensitive values.
- Keep changes scoped and run `go test ./...` from `backend/` after Go changes.
