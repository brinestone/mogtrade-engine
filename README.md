# MogTrade - Trading Platform Engine

MogTrade is a Go-based trading platform engine that provides a RESTful API for frontends and clients to place trade orders on connected stock market exchanges. It features a modular architecture separated into four main packages: `app`, `core`, `infra`, and `services`.

## Core Features

- **RESTful API** using GIN framework for order placement, account management, and market data
- **SQLC-generated domain models and query functions** for type-safe database operations
- **Asynq job queue** replacing CronJobScheduler for background task processing
- **Multi-exchange data feeds** via Massive and Alpha Vantage APIs
- **JWT authentication** with token encoding/verification and role authorization
- **Sentry error tracking** and structured logging via `slog`
- **Prometheus/Grafana monitoring** integrations
- **Database migration system** with sqlc code generation (use `mogtrade-migrate` MCP tools)
- **Order matching engine** in `services/matching/` with execution creation and fill logic
- **Wallet & escrow management** with event-driven fill payouts

## Project Structure

| Package | Purpose |
|---|---|
| `app/` | Main entrypoint, IoC Container, GIN API, controllers, middleware |
| `core/` | Library interfaces and abstractions (contracts, models) |
| `infra/` | External integrations: DB, exchanges, job queues, storage, mail, events |
| `services/` | Business logic: matching, orders, billing, auth, kyc, market, payments |
| `go.work/` | Go workspace linking all modules |

## Key Modules

- **Matching engine** (`services/matching/`) — Order book, execution creation, `ExecutionCreatedEventArgs`
- **Wallet & escrow** (`app/controller/wallet.go`) — Fund management, `fillPendingExecutions`, event handler
- **Asynq job queue** (`infra/jobs/`) — `AsynqJobQueue` with Redis, replaces `CronJobScheduler`
- **Data feeds** (`infra/feed/`) — `MassiveDatasource`, `AlphaVantageDatasource` implementing `feed.Datasource`
- **Auth & security** (`app/controller/auth_confirm.go`, `core/contract/`) — JWT, role authorizer, bearer claims

## Development

### Build & Test

```bash
go build ./...
go test ./...
```

### Code Conventions

- Use `any` type (not `interface{}`)
- Use `new(any)` to obtain pointers
- Conventional commits: `type(scope): description`
- Branches: `feat/<feature>`, `fix/<bug>`, `refactor/<purpose>`
- PRs against `main` after approval

### Ignored Files

- `.vfox/` — SDK symlinks
- `app/cmd/.env` — Secret values (example: `.env.example`)
- All files in `.gitignore`

### Database Migrations

**Use the `mogtrade-migrate` MCP server tools for all schema changes.** Never write migration queries manually. The migration sequence `000050→000055` is cumulative and coherent.

## Environment Variables

Required vars (example in `app/cmd/.env.example`):

- `MASSIVE_API_KEY` — Massive API key for market data
- `ALPHAVANTAGE_API_KEY` — Alpha Vantage API key
- `DB_URL` — PostgreSQL connection string
- `JWT_SECRET` — JWT signing secret
- `PORT` — Server port
- `REDIS_URL` — Redis connection for Asynq job queue
- `S3_PLATFORM`, `S3_ACCESS_KEY`, `S3_SECRET`, `S3_ORIGIN`, `S3_REGION` — Storage configuration
- `MAILTRAP_API_KEY` — MailTrap SMTP service
- `EXCHANGE_RATE_API_KEY` — Currency conversion API
- `ALLOWED_ORIGINS` — CORS origins (semicolon-separated)
- `HOST` — Host address for token encoding