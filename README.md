
# Pehlione — Go SSR E‑Commerce Platform

A Go-based, server-side rendered e‑commerce platform with a web server and background workers (email/sms/outbox), opinionated templates, and an admin surface.

## Overview

Pehlione implements a full e‑commerce workflow:
- Web server (`cmd/web`) built with Gin serving type-safe `templ` templates (SSR).
- Background workers (`cmd/worker`, `sms-worker`) for email outbox and other async jobs.
- MySQL-backed data model with migrations, payment abstraction, PDF invoice generation, and admin order management.

## Features

- Session-based authentication and guest checkout
- Role-based authorization (user / admin)
- CSRF protection (double-submit cookie pattern)
- Cookie-based cart (guest) and DB-based cart (authenticated)
- Checkout with idempotency and stock reservation (pessimistic locking)
- Async email outbox + MailHog support for local testing
- Payment intent tracking and refund processing (webhook-aware)
- PDF invoice generation (go-pdf/fpdf)
- Accessibility and performance optimizations (ARIA, lazy images, async decoding)
- Structured logging and request ID tracking
- Template component architecture (StandardProductCard, SaleProductCard)

## Tech Stack

- Go (module: [pehlione.com/go.mod](pehlione.com/go.mod) — Go 1.25.1)
- Gin web framework ([github.com/gin-gonic/gin])
- GORM ORM with MySQL driver
- Templ for type-safe templates ([github.com/a-h/templ])
- Tailwind CSS + PostCSS (frontend build via `package.json`)
- Mage for developer tasks (`magefile.go`)
- Goose for DB migrations
- Docker + docker-compose for local containers

## Repository Structure (key folders)

pehlione.com/
- `cmd/` — `web`, `worker` entrypoints
- `internal/` — `http/` handlers & middleware, `modules/` (auth, cart, checkout, email, orders, payments), `pdf/`
- `pkg/` — view models
- `templates/` — `components/`, `layout/`, `pages/`
- `static/` — built frontend assets
- `migrations/` — goose migrations
- `magefile.go`, `Dockerfile`, `docker-compose.yml`

Example tree:
```
pehlione.com/
├── cmd/
│   ├── web/
│   └── worker/
├── internal/
├── templates/
├── static/
├── migrations/
└── magefile.go
```

## Prerequisites

- Go >= 1.25 (module lists `go 1.25.1`)
- MySQL 8.0+ (compose uses `mysql:8.0`)
- Node.js + npm (Tailwind/PostCSS)
- Optional tools: `templ`, `air`, `golangci-lint`, `goose` (Mage can install some tools)

## Quickstart

Clone and prepare:
```bash
git clone https://github.com/1DeliDolu/pehlione_go.git
cd pehlione.com
go mod download
npm install
```

Generate templates:
```bash
templ generate
# or for watch mode:
templ generate --watch
```

Run DB migrations (example):
```bash
# Using goose directly (example from docs)
goose -dir migrations mysql "user:pass@/pehlione_go" up

# Or using mage wrapper (requires DB_DSN env set)
mage MigrateUp
```

Run the app (development):
```bash
# Hot reload (if `air` is installed)
mage Dev

# Fallback to go run
mage Run
# or
go run ./cmd/web
```

Build production binary:
```bash
mage Build
# output in ./bin (e.g., ./bin/pehlione-web or ./bin/pehlione-web.exe on Windows)
```

Run with docker-compose:
```bash
docker compose up --build
# or
docker-compose up --build
```

## Local Development

Install helper tools with Mage:
```bash
mage Tools
```

Common Mage targets:
- `mage Dev` — dev server (uses `air` if available)
- `mage Gen` — `templ generate`
- `mage Build` — build `cmd/web` binary
- `mage Test` — `go test ./... -count=1`
- `mage Fmt` — `gofmt`
- `mage Lint` — `golangci-lint run`
- `mage CSS` / `mage CSSWatch` — build/watch Tailwind via npm scripts

Frontend build (npm scripts in `package.json`):
```bash
npm run build:css   # postcss -> ./static/css/app.css
npm run dev:css     # watch mode
```

### Environment Variables / Configuration

Common environment variables referenced in repo and `docker-compose.yml`:
- `DB_DSN` — e.g. `root:@tcp(db:3306)/pehlione_go?parseTime=true`
- `SECRET_KEY` — application secret for sessions/cookies
- `SESSION_TTL_HOURS` — session lifetime
- `GIN_MODE` — `release`/`debug`
- SMTP: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_USE_TLS`

The docs reference copying `.env.example` to `.env`. If `.env.example` is not present, set env vars manually or in your container.

## Testing

Run tests:
```bash
mage Test
# or
go test ./... -count=1
```

Race-enabled tests:
```bash
mage TestRace
```

## Linting / Formatting

Format and lint:
```bash
mage Fmt
mage Lint
# If missing, install tools:
mage Tools
```

## Docker

The repository includes:
- `Dockerfile` — multi-stage build producing `/app/web` and `/app/worker` binaries (EXPOSE 8080).
- `docker-compose.yml` — defines `db` (MySQL), `mailhog`, `web`, `worker`, `sms-worker`.

Ports mapped in `docker-compose.yml`:
- MySQL: `3306:3306`
- Web: `8080:8080`
- MailHog UI: `8025:8025`, SMTP: `1025:1025`

Example: to start full stack locally
```bash
docker compose up --build
```

## API (discoverable endpoints from docs)

Public:
- `GET  /` — Home
- `GET  /products` — Product listing
- `GET  /cart` — Cart page
- `POST /cart/add` — Add to cart (CSRF)
- `GET  /checkout` — Checkout page
- `POST /checkout` — Create order (CSRF)
- `GET/POST /signup`, `GET/POST /login`, `POST /logout`

Authenticated:
- `GET  /account/orders`
- `GET  /orders/:id`
- `POST /orders/:id/pay`

Admin:
- `GET /admin/orders`
- `GET /admin/orders/:id`
- `POST /admin/orders/:id`

Example quick curl:
```bash
curl -v http://localhost:8080/products
```

## Troubleshooting

- Missing `templ`: run `mage Tools` or install `github.com/a-h/templ/cmd/templ`.
- Missing `air`: `mage Dev` falls back to `go run ./cmd/web`.
- Migration failures: verify `DB_DSN` and MySQL availability. Use `mage MigrateUp` after setting `DB_DSN`.
- CSS stale: run `npm run build:css` or `mage CSS`.

## Contributing

1. Fork the repository
2. Create branch: `git checkout -b feature/YourFeature`
3. Run `mage Fmt`, `mage Lint`, `mage Test` locally
4. Commit & push, open a Pull Request

## Security

Implemented protections documented in project:
- CSRF double-submit cookie pattern
- Bcrypt password hashing
- Session cookies with `HttpOnly` and `SameSite=Lax` (`Secure` required in production)
- Parameterized DB queries (GORM) and template auto-escaping
- Input validation via `go-playground/validator`

## License

MIT — see `LICENSE`.

---

## Assumptions / TODO

- Verified files scanned: `pehlione.com/go.mod`, `pehlione.com/Dockerfile`, `pehlione.com/docker-compose.yml`, `pehlione.com/package.json`, `pehlione.com/magefile.go`, and repository README content. References to these files appear in this README (paths shown).
- `.env.example` was referenced in project docs but its presence/content was not confirmed during the scan — add a `.env.example` with required vars if missing.
- CI workflows (e.g., `.github/workflows`) were not found during inspection — add CI docs or badges if you maintain pipelines.
- Seed scripts and explicit database seed commands were not located; test user entries referenced in docs likely come from migrations or seeds not present in the scanned files.
- OpenAPI/Swagger or machine-readable API docs were not found — consider adding API schema if needed.
