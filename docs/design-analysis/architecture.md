# architecture.md

This document serves as the central architecture reference for the MogTrade Engine, linking to detailed concept documents for each major component.

## Table of Contents

1. [Module Layering & Dependency Rules](#module-layering--dependency-rules)
2. [IoC Container & GIN Server](#ioc-container--gin-server)
3. [API Layer Structure](#api-layer-structure)
4. [Market Data & Feed Ingestion](#market-data--feed-ingestion)
5. [Order Management & Matching](#order-management--matching)
6. [Background Jobs & Scheduling](#background-jobs--scheduling)
7. [Infrastructure & External Integrations](#infrastructure--external-integrations)

## 1. Module Layering & Dependency Rules

The project follows a strict layered architecture with clear dependency rules:

- **app**: Application layer containing main entrypoint, IoC container, and server bootstrap
- **core**: Core library containing interfaces and abstractions
- **services**: Business logic implementing core interfaces
- **infra**: Infrastructure layer providing concrete implementations

**Dependency Direction**: app → core → services → infra (outer layers depend on inner layers)

## 2. IoC Container & GIN Server

The `app` module contains the main application structure:

- **IoC Container**: Located in `app/cmd/ioc.go`, uses `go-slim.dev/ioc` for dependency injection
  - Manages database connections (pgx pool)
  - Instantiates service layer components
  - Provides API server configuration
  - Manages background job scheduling

- **GIN Server**: Main entrypoint in `app/cmd/main.go` that:
  - Parses environment variables
  - Sets up structured logging (JSON + text handlers)
  - Initializes IoC container
  - Mounts API routes
  - Starts background tasks (market feed, job scheduler)
  - Handles graceful shutdown

## 3. API Layer Structure

The API layer (`app/controller`, `app/payloads`, `app/middleware`) follows clean separation:

- **Controllers**: Thin wrappers that:
  - Parse HTTP requests
  - Validate input payloads
  - Call service layer methods
  - Return HTTP responses
  - Emit domain events

- **Payloads**: DTOs for request/response:
  - HTTP request payloads in `app/payloads/http/`
  - Event payloads in `app/payloads/events/`
  - Error responses in `app/payloads/http/errors.go`

- **Middleware**: Cross-cutting concerns:
  - `auth.go`: JWT authentication middleware
  - `rate-limiter.go`: Rate limiting middleware
  - `helpers.go`: Request context helpers

## 4. Market Data & Feed Ingestion

Market data is processed through a feed abstraction:

- **Feed Interface**: `core/feed/feed.go` defines `Datasource` with `Pull()` method
- **Data Sources**:
  - `MassiveDatasource` (core/feed/massive.go): New implementation for aggregate bar data
  - `AlphaVantageDatasource`: Existing tick data implementation
- **Poller**: `ExchangePoller` (services/market/exchange-poller.go) polls data sources
- **Event Hub**: `ExchangeHub` (services/market/exchange-hub.go) broadcasts events

## 5. Order Management & Matching

Order lifecycle is managed through:

- **Order Model**: `services/orders/service.go` defines order parameters and execution params
- **Risk Engine**: `services/orders/risk-engine.go` validates orders against business rules
- **Matching Engine**: `services/matching/engine.go` handles order matching logic
- **Order Book**: `services/matching/book.go` manages bid/ask books with mutex locks

### Matching Method Implementation

The `findMatches()` method in `services/matching/engine.go` needs implementation to:

1. Get best bid and ask prices from order books
2. Calculate match quantity as minimum of available quantities
3. Match orders at best available price
4. Update order quantities and remove completed orders
5. Emit match events via ExchangeHub

Key considerations:
- Lock ordering to prevent deadlocks
- Time priority for same-price orders
- Partial match handling
- Thread-safe operations

## 6. Background Jobs & Scheduling

Background tasks are managed through:

- **Job Interface**: `core/contract/jobs.go` defines `Job` with `Schedule()` and `Run()` methods
- **Job Scheduler**: `core/contract/jobSchedular.go` provides `RegisterJob()` and `Start()` methods
- **Job Implementations**:
  - `StaleTokenRemoverJob` (services/jobs/refresh-token-cleaner.go) - weekly scheduled job
  - Other jobs follow same pattern

- **Scheduling**: Jobs registered in `setupBackgroundJobs()` (ioc.go) using cron-like syntax

## 7. Infrastructure & External Integrations

Infrastructure layer provides:

- **Database**: SQLC-generated models with pgx integration
- **Event Bus**: In-memory event bus in `infra/events/bus.go`
- **Mail Service**: Mailtrap integration in `infra/mail/`
- **Id Generation**: ULID-based ID generation in core
- **Auth**: JWT-based authentication with claims validation

## Central Reference

All concept documents are stored in `.pi-herdsman/design-analysis/` and linked from this central architecture.md document.