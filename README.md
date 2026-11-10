# Ayaka Backend Template

Ayaka is a production-ready backend API template written in Go, following Clean (Hexagonal) Architecture. It gives you authentication, structured logging, migrations, validation, consistent error handling, tests, Docker and CI out of the box, so you can start writing business logic on day one.

## Technical Specifications

* **Language:** [Go](https://go.dev/) 1.25
* **Web Framework:** [Fiber v2](https://gofiber.io/)
* **Database:** PostgreSQL + [GORM](https://gorm.io/)
* **Migrations:** [goose](https://github.com/pressly/goose) (SQL files embedded in the binary)
* **Configuration:** Viper + godotenv (YAML with `${ENV:default}` interpolation, validated at startup)
* **Logging:** `log/slog` (JSON/text), request-scoped with `request_id` / `user_id`
* **Auth:** JWT access + refresh tokens (HS256, separate secrets, typed tokens)
* **Validation:** go-playground/validator v10 with DB-aware custom rules
* **API Docs:** Swagger (swaggo)
* **Tooling:** Makefile, golangci-lint v2, air (hot reload), govulncheck, Dependabot
* **Containerization:** Multi-stage Docker build (static binary, non-root) + Docker Compose
* **CI:** GitHub Actions (lint, race-enabled tests with coverage, vulnerability scan, Docker build)

## Features

1. **Strict layered architecture** — `core` (entities, use cases, ports) has no dependency on HTTP or GORM. Adapters implement the ports; the composition root wires them.
2. **Ready-made auth module** — register, login, refresh, `GET /users/me`, role-based access (`admin`-only paginated user list).
3. **Centralized error handling** — one `apperror` type with a transport-agnostic `Kind`, mapped to HTTP in exactly one place. Internal error details are logged, never leaked to clients.
4. **Observability** — request IDs (propagated via `X-Request-ID`), one structured access log per request, panic recovery with stack traces.
5. **Security defaults** — CORS allow-list, security headers (helmet), rate limiting on `/auth`, body size limit, timing-safe login, generic credential errors, refuses to start with weak JWT secrets outside development.
6. **Health probes** — `/livez` (liveness) and `/readyz` (readiness, pings the DB); `/health` kept as an alias.
7. **Versioned migrations** — SQL migrations with `migrate up|down|status|create`, optional `serve --migrate`.
8. **Transactions** — `TxManager` propagates a transaction through `context.Context`; repositories pick it up automatically (nested calls reuse the outer transaction).
9. **Pagination** — type-safe `model.Page[T]` in the core and a standard `meta` block in responses.
10. **Graceful shutdown** — stops accepting traffic, drains in-flight requests, closes the DB pool.

## Project Structure

```text
be-ayaka/
├── .github/                 # CI workflow + Dependabot
├── cmd/                     # CLI: serve, migrate (cobra)
├── config/                  # Config loading & validation
├── docs/                    # Generated Swagger docs (make swag)
├── migrations/              # SQL migrations (embedded into the binary)
├── internal/
│   ├── adapter/             # Infrastructure (implements core ports)
│   │   ├── database/        # PostgreSQL connection pool + migrator
│   │   ├── email/           # SMTP sender
│   │   ├── repository/      # GORM repositories + TxManager
│   │   └── token/           # JWT token provider
│   ├── bootstrap/           # Composition root: container, server, routes, lifecycle
│   ├── core/                # Business logic — framework agnostic
│   │   ├── apperror/        # Application error type & kinds
│   │   ├── entity/          # Persistent domain entities
│   │   ├── model/           # Use-case inputs/outputs (incl. pagination)
│   │   ├── port/            # Interfaces the core depends on
│   │   └── service/         # Use cases
│   ├── delivery/http/       # Fiber handlers (thin) + dto/ (request/response payloads)
│   ├── middleware/          # Auth, roles, request context/logging, errors, security
│   └── testingutils/        # Test helpers and mocks
├── pkg/                     # Reusable, domain-agnostic packages (hash, logger, validator, ...)
├── config.yaml              # Configuration with ${ENV:default} placeholders
├── Makefile                 # Developer workflow (make help)
└── main.go
```

### Request flow

```text
HTTP ─► middleware (request id → logging/errors → recover → security → auth)
     ─► handler (bind + validate DTO → model)
     ─► service (business rules, transactions)
     ─► port interface ─► adapter (GORM / JWT / SMTP)
errors bubble up as apperror ─► ErrorHandler ─► JSON envelope + log
```

## Core Concepts

### Kind-based application errors (`internal/core/apperror`)
Every error carries a **`Kind`** (what went wrong) instead of an HTTP status code (how to show it):

```go
// core / service: describes the problem, knows nothing about HTTP
var ErrPostNotFound = apperror.NotFound("post not found")   // Kind = KindNotFound

// delivery: the only place that maps Kind -> HTTP (internal/middleware/error_handler.go)
apperror.KindNotFound: fiber.StatusNotFound,               // -> 404 DATA_NOT_FOUND
```

Why:
* **The core stays transport-agnostic** — the same service can be reused by gRPC, a queue worker or a CLI, each with its own Kind mapping.
* **One mapping table** — changing how an error is rendered is a one-line change, not a hunt through handlers.
* **Broad or specific checks** — `errors.Is(err, apperror.ErrNotFound)` matches *any* not-found error; `errors.Is(err, service.ErrInvalidCredentials)` matches only that sentinel.
* **Safe by default** — the cause (`.Wrap(err)`) is logged but never sent to the client; unknown errors become `500 internal server error`.

Cheat sheet:

| Need | Use |
| :--- | :--- |
| Reusable domain error | `var ErrX = apperror.Conflict("...")` |
| Unexpected failure with cause | `return apperror.Internal(err)` |
| Per-field details | `apperror.Validation(map[string]string{"field": "msg"})` |
| Keep message, add cause | `ErrX.Wrap(err)` |

### Composition root (`internal/bootstrap/container.go`)
The container **assembles** the application: it creates the adapters (repositories, JWT provider, hasher), injects them into services, and injects services into handlers. It is the **only** place that knows concrete types — everything else depends on interfaces from `core/port`. Compile-time assertions (`var _ port.UserRepository = (*repository.UserRepo)(nil)`) make the build fail if an adapter stops satisfying its port.

```text
container.go:  repository ─► service ─► handler      (wiring)
routes.go:     handler ─► URL + middleware            (exposure)
bootstrap.go:  DB + Fiber + container + routes ─► App (lifecycle: run & graceful shutdown)
```

## API

| Method | Path | Auth | Description |
| :--- | :--- | :--- | :--- |
| GET | `/livez` | – | Liveness probe |
| GET | `/readyz` (`/health`) | – | Readiness probe (checks DB) |
| GET | `/swagger/index.html` | – | Swagger UI (when `SWAGGER_ENABLED=true`) |
| POST | `/api/v1/auth/register` | – | Create an account |
| POST | `/api/v1/auth/login` | – | Get access + refresh tokens |
| POST | `/api/v1/auth/refresh` | – | Exchange a refresh token for a new pair |
| GET | `/api/v1/users/me` | Bearer | Current user profile |
| GET | `/api/v1/users?page=1&pageSize=20` | Bearer (admin) | Paginated user list |

### Response envelope

```json
{
  "status": "SUCCESS",
  "message": "users retrieved",
  "data": [ ... ],
  "meta": { "totalRecords": 42, "totalPages": 3, "currentPage": 1, "pageSize": 20, "hasNext": true, "hasPrevious": false },
  "requestId": "REQ-01932c7e6f2a7c3b9f1d2e3a4b5c6d7e"
}
```

Errors use the same envelope; validation errors list fields in `errorField`:

```json
{
  "status": "UNPROCESSABLE_ENTITY",
  "message": "validation failed",
  "errorField": { "email": "Email must be a valid email address" },
  "requestId": "REQ-..."
}
```

| Error kind (`apperror`) | HTTP | `status` |
| :--- | :--- | :--- |
| `KindBadRequest` | 400 | `BAD_REQUEST` |
| `KindUnauthorized` | 401 | `UNAUTHORIZED` |
| `KindForbidden` | 403 | `FORBIDDEN` |
| `KindNotFound` | 404 | `DATA_NOT_FOUND` |
| `KindConflict` | 409 | `DATA_CONFLICT` |
| `KindValidation` | 422 | `UNPROCESSABLE_ENTITY` |
| `KindTooManyRequests` | 429 | `TOO_MANY_REQUESTS` |
| `KindInternal` / unknown | 500 | `SERVER_ERROR` |

## Getting Started (Local Development)

### 1. Prerequisites
Go 1.25+, PostgreSQL 14+ and `make` (on Windows: Git Bash/WSL, or `choco install make`).

### 2. Configure
```bash
cp .env.sample .env   # then edit the values
```
`JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET` are **required**, must differ, and must be at least 32 characters outside `development` (`openssl rand -base64 48`). Durations use Go syntax (`15m`, `168h`). See [config.yaml](config.yaml) for every option.

### 3. Create the schema
```bash
createdb ayaka_db       # or create it with your favourite tool
make migrate-up
```

### 4. Run
```bash
make run    # or: make dev  (hot reload)
```
Then open `http://localhost:8000/readyz` and `http://localhost:8000/swagger/index.html`.

> To create the first admin, register a user and promote it:
> `UPDATE users SET role = 'admin' WHERE email = 'you@example.com';`

## Getting Started (Docker Compose)

```bash
cp .env.sample .env
docker compose up --build    # or: make up
```
Compose waits for PostgreSQL to be healthy, then starts the app with `serve --migrate`, which applies pending migrations before serving traffic.

## Development Workflow

### Adding a feature
1. **Migration** — `make migrate-create name=create_posts_table`, write the SQL, `make migrate-up`.
2. **Entity** — `internal/core/entity/` (persistent struct, keep gorm tags in sync with the SQL).
3. **Port** — repository interface in `internal/core/port/`.
4. **Use case** — inputs/outputs in `internal/core/model/`, logic in `internal/core/service/`. Return `apperror` values (e.g. `apperror.NotFound("post not found")`).
5. **Adapter** — repository in `internal/adapter/repository/` (use `conn(ctx, r.db)` so transactions work, and `translate()` for errors).
6. **Delivery** — DTOs in `internal/delivery/http/dto/`, handler in `internal/delivery/http/` (bind → service → DTO; just `return err`).
7. **Wire** — add it to `internal/bootstrap/container.go` and a route in `routes.go`.
8. **Docs & tests** — add swag annotations, run `make swag`, write tests (see [TESTING.md](TESTING.md)).

### Conventions
* Handlers never log errors — they return them; the global `ErrorHandler` logs 5xx with the cause and renders the response.
* Log with `logger.FromContext(ctx)` so every line carries `request_id` (and `user_id`).
* Pass `c.UserContext()` (not `c.Context()`) from handlers into services.
* Never expose entities directly in responses — map them to DTOs.

### Before pushing
```bash
make check   # fmt + lint + test (the same checks CI runs)
```

## Custom Validation Tags

```go
type RegisterRequest struct {
    // unique=table->column : value must not exist yet (soft-deleted rows are ignored) → 409
    Email    string `json:"email" validate:"required,email,unique=users->email"`
    // incolumn=table->column : value must exist (FK check) → 404
    RoleID   string `json:"roleId" validate:"required,incolumn=roles->id"`
    // username : letters, numbers, underscores, hyphens and dots
    Username string `json:"username" validate:"required,username"`
    // complexpassword : 8-64 chars, upper, lower, number and special character
    Password string `json:"password" validate:"required,complexpassword"`
    // whitespace : rejects any whitespace
    Code     string `json:"code" validate:"whitespace"`
}
```
DTOs may implement `Normalize()` (trim/lowercase) which runs before validation. Rules live in [pkg/validator/validator.go](pkg/validator/validator.go).

> `unique` gives friendly errors, but the database unique index is the real guarantee (it also covers concurrent requests); repositories map unique violations to `409 Conflict`.

## Upgrading From the Previous Version

If you started from the earlier version of this template:

| Area | Before | Now | Action |
| :--- | :--- | :--- | :--- |
| JWT durations | `JWT_ACCESS_EXPIRED='24'` (minutes) | Go durations: `'15m'`, `'168h'` | Update `.env` — plain numbers fail to start |
| JWT secrets | Defaults (`secret_default`) | Required, must differ, ≥ 32 chars outside development | Set both secrets in `.env` |
| Server port | `HTTP_PORT` in `config.yaml` | `SERVER_PORT` everywhere | Use `SERVER_PORT` |
| Schema | `AutoMigrate` on startup | goose SQL migrations | Run `make migrate-up` (or `serve --migrate`) |
| `users` table | No `display_name`, non-partial unique constraints | Created by `migrations/…_create_users_table.sql` | Dev DB: `DROP TABLE users;` then `make migrate-up`. Existing data: write a migration that adds `display_name` and the partial unique indexes |
| Profile endpoint | `GET /api/v1/user/profile` | `GET /api/v1/users/me` | Update clients |
| Errors | `customerrors` + long switch in the error handler | `apperror` + Kind mapping | Replace `customerrors.ErrX` with `apperror` sentinels |
| Logging | `logger.Log("SYSTEM", "INFO", msg, requestId)` | `logger.FromContext(ctx).Info(msg, "key", val)` | Use the context logger |
| Responses | `errorField` built by parsing strings | `errorField` is a `map[string]string` | No client change (same JSON shape) |
| Run command | `go run main.go svc` | `go run . serve` (`svc` still works) | — |

## Known Limitations / Next Steps
* Refresh tokens are stateless (no revocation/logout list). For logout or token rotation detection, store token IDs (`jti`) in Redis or a table.
* The rate limiter uses in-memory storage; use a shared store (e.g. Redis) when running multiple instances.
* Repository tests use in-memory SQLite; add PostgreSQL integration tests (e.g. testcontainers-go) for Postgres-specific behaviour.

## Credits & Acknowledgements

* **Hanashiro Yuriku | Dionisius Geovanni Caesario**
* **Gemini** - *AI Pair Programmer*
  Served as a technical consultant and coding assistant, providing architectural insights and collaborative debugging throughout the development journey.
* **Claude (Anthropic, Claude Opus 5.5)** - *AI Pair Programmer*
  Reviewed the template and implemented the production-readiness overhaul: transport-agnostic error model, auth module, SQL migrations, structured request logging, security middleware, test suite, and the Docker/CI/Makefile developer workflow.

  **Disclaimer:** *The names Ayaka used in this architectural blueprint are inspired by characters from Genshin Impact. This project is open-source, purely for personal/educational purposes, and is not affiliated with, endorsed, or sponsored by HoYoverse/Cognosphere.*
