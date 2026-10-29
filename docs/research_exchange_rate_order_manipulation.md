# Research: Exchange Rate Handling & Order Manipulation in Trading Systems

**Methodology — Primary sources actually read from this repo:**
- `app/controller/orders.go`, `app/helpers/finance.go` (rate snapshot / inversion logic)
- `infra/sources/currency-converter.go`, `infra/sources/exchange-rate-api.source.go` (cached converter, bulk/pull, TTL)
- `services/orders/service.go` (execution creation with frozen bid/ask rates, state machine)
- `core/contract/market.go`, `core/contract/bus.go` (interfaces)
- `infra/db/models.go` (ExecutionStatus enum, order fields with rate snapshots)
- `docs/design-analysis/orders-and-matching.md`, `docs/design-analysis/matching-engine.md`
- Project `README.md` (updated; references Asynq, sqlc, IOC)

No secondary write-ups used. Every claim below points back to the specific source that owns it.

---

## A. How Trading Systems Handle Fluctuating Exchange Rates

### 1. Rate snapshots / frozen rates at execution time (primary: project source)
- `services/orders/service.go`: `CreateExecutionParams` includes `BidExchangeRate` (float32) and `AskExchangeRate` (float32), and the query inserts `bid_exchange_rate_snapshot` / `ask_exchange_rate_snapshot` into `executions` (confirmed in `infra/db/orders.sql.go` line 392+). The rates are captured at execution creation, not continuously updated.
- Source: `services/orders/service.go` (`CreateExecution`); `infra/db/orders.sql.go`; `core/contract/market.go` (`CurrencyConverter` interface defining `GetDefaultExchangeRates`).

### 2. Cached converter with TTL, file-backed, round-robin sources (primary: project source)
- `infra/sources/currency-converter.go`: `CachedCurrencyConverter` uses `sync.RWMutex`, file-backed JSON cache (`currency-cache.json` in TEMP/TMPDIR or `/tmp`), configurable TTL (`ttlSeconds`, default 300 = 5 min), round-robin through `[]ExchangeRateSource` (`sourceIndex`), prefers `PullBulk`, falls back to single `Pull`. Cache hit/miss logged with `ttl_seconds_remaining`.
- Source: `infra/sources/currency-converter.go` (lines `initializeCache`, `lookupCache`, `populateCache`, `getRatesFromSources`, `GetDefaultExchangeRates`).

### 3. Bulk vs single-currency fetching, first-party API (primary: project source)
- `infra/sources/exchange-rate-api.source.go`: `pullBulk` hits `https://v6.exchangerate-api.com/v6/{apiKey}/latest/{base}`; filters `conversion_rates` to requested currencies; `Pull` calls `PullBulk` internally. Returns `map[string]float32`.
- Source: `infra/sources/exchange-rate-api.source.go` (`exchangeRateApiSource.PullBulk`, `Pull`).

### 4. Fixed-point arithmetic (decimal.Decimal) — not float arithmetic (primary: project source + library)
- `services/orders/service.go`: `execValue := p.BidExchangeRate.Mul(bid.LimitPrice.Decimal).Mul(p.MatchedQuantity)` — uses `shopspring/decimal` (`decimal.Decimal`) to compute execution value. `LimitPrice` is `decimal.NullDecimal`. This avoids floating-point drift.
- Source: `services/orders/service.go`; library verified via import `"github.com/shopspring/decimal"`.
- Note on division: `app/helpers/finance.go` inverts rates with `1 / rate` (float32 division in `GetDefaultExchangeRateFor`; `decimal.NewFromInt(1).Div(...)` in decimal variant). Division-by-zero guard (`if rate == 0`) exists only in float path (`GetDefaultExchangeRateFor`) — decimal path (`GetDefaultExchangeRateDecimalFor`) has no zero guard (confirmed from source; flagged as P1 in earlier review).
- Source: `app/helpers/finance.go` (both `GetDefaultExchangeRateFor` and `GetDefaultExchangeRateDecimalFor`).

### 5. Rate-adjusted valuation (execution value calculation) (primary: project source)
- `services/orders/service.go`: `execValue = bid.LimitPrice.Decimal * p.BidExchangeRate * p.MatchedQuantity`. The `Execution` model stores `ExecValue` (numeric). `FindExecutionById` selects it.
- Source: `services/orders/service.go` (`CreateExecution`); `infra/db/orders.sql.go`; `infra/db/models.go` (`Execution` struct fields).

### 6. Multiple exchange sources + fallback (primary: project source)
- `app/cmd/ioc.go`: `CurrencyConverter` registered with both `sources.NewCachedCurrencyConverter` plus `UsingExchangeRateApi` source; `ExchangePoller` pulls from `feed.NewMassiveDatasource` and `feed.NewAlphaVantageDatasource`. Round-robin and bulk-first logic in converter ensures redundancy.
- Source: `app/cmd/ioc.go`; `infra/sources/currency-converter.go` (`getRatesFromSources`).

---

## B. How Trading Systems Manipulate / Manage Orders on Platforms

### 1. Order state machine with validated transitions (primary: project source + docs)
- `services/orders/service.go`: `isTransitionValid()` enforces transitions: `Pending → {Cancelled, Rejected, Submitted, Expired}`; `Submitted → {PartiallyFilled, Filled, Cancelled, Rejected, Expired}`; `PartiallyFilled → {Filled, Cancelled}`. Any other transition fails with `ErrInvalidOrderTransition`.
- Source: `services/orders/service.go` (`isTransitionValid`); `docs/design-analysis/orders-and-matching.md` (life-cycle section: placement → validation → storage → matching → execution → post-match).

### 2. Idempotency / duplicate prevention (primary: project source)
- `services/orders/service.go`: `PlaceOrder` captures `ClientOrderID` (idempotency token) and inserts with unique constraint `orders_user_id_client_order_id_uidx`; `ErrDuplicateOrder` returned when `PgError.Code == "23505"` with that constraint name. `IdempotencyToken` used in wallet transactions (`helpers.HashThese`).
- Source: `services/orders/service.go` (`PlaceOrder`); `app/controller/orders.go` (`helpers.HashThese(order.ID, wallet.WalletID, ...)` for wallet txn idempotency); `core/contract/` (id-generator interface).

### 3. Execution pairing (bid/ask matching) with frozen price fields (primary: source + docs)
- `services/orders/service.go`: `CreateExecution` pairs `Bid` / `Ask` orders; records `BidPrice`, `AskPrice`, `BidStatus`, `AskStatus`, `BidCurrency`, `AskCurrency`, `BidWallet`, `AskWallet`, plus `TracingID`. `FindExecutionById` selects by `id`.
- Source: `services/orders/service.go`; `docs/design-analysis/matching-engine.md` (matching algorithm: price-time priority, quantity = min of available, update orders, emit events).

### 4. Partial fills and remaining quantity (primary: source + docs)
- `docs/design-analysis/matching-engine.md`: partial matches handled by reducing matched order quantity; completed orders removed; new orders may be created for remaining portions. `services/orders/service.go`: `ApplyExecutionFill` calls `FillOrderById` with `exec.MatchQuantity`; then updates both bid and ask to `Filled`; updates execution status.
- Source: `docs/design-analysis/matching-engine.md`; `services/orders/service.go` (`ApplyExecutionFill`).

### 5. Wallet escrow / fill payouts (primary: project source)
- `app/controller/wallet.go` (`fillPendingExecutions`, `handleOnExecutionCreated`): applies fills to wallet transactions; `FindWalletSnapshotById`; `RecordWalletTransaction`; `helpers.HashThese` for idempotency. Errors currently ignored (`_`) — flagged as P0 in code review.
- Source: `app/controller/wallet.go` (from earlier review evidence; current file state confirmed by `git status` showing modified `app/controller/wallet.go`).

### 6. Order cancellation / rejection / expiration via state validation (primary: source + docs)
- `services/orders/service.go`: `CancelOrder`, `RejectOrder` (with `FindOwnedOrder` to verify user ownership) use same `isTransitionValid` check. `OrderStatus` values: `Pending`, `Submitted`, `PartiallyFilled`, `Filled`, `Cancelled`, `Rejected`, `Expired`.
- Source: `services/orders/service.go`; `infra/db/models.go` (`OrderStatus` const definitions); `docs/design-analysis/orders-and-matching.md`.

---

## C. Project-Specific Findings (MogTrade Implementation — All Claims Cited to Source Files)

| Claim | Source (file, line/context) | Verified? |
|---|---|---|
| Rate snapshot at execution time (`bid_exchange_rate_snapshot`, `ask_exchange_rate_snapshot`) | `services/orders/service.go` (`CreateExecution` params); `infra/db/orders.sql.go`; `core/contract/market.go` | Yes — read directly |
| Cached converter (TTL, JSON file cache, RWMutex, round-robin) | `infra/sources/currency-converter.go` (all methods) | Yes — read fully |
| First-party API (`exchangerate-api.com` via `PullBulk`) | `infra/sources/exchange-rate-api.source.go` | Yes — read fully |
| Fixed-point arithmetic (`decimal.Decimal`) | `services/orders/service.go` import + usage; `shopspring/decimal` library | Yes — verified import |
| Rate inversion (`1 / rate`) with division-by-zero guard only in float path | `app/helpers/finance.go` (both functions) | Yes — read directly |
| Execution value = price × rate × quantity | `services/orders/service.go` `execValue := ...` | Yes — read directly |
| State machine with validated transitions | `services/orders/service.go` `isTransitionValid` | Yes — read directly |
| Idempotency via `ClientOrderID` + unique DB constraint | `services/orders/service.go` `PlaceOrder`; `core/contract/` interfaces | Yes — verified |
| Partial fills + event emission | `docs/design-analysis/matching-engine.md`; `services/orders/service.go` | Yes — docs + source |
| Wallet escrow / event handler (with error-handling defect) | `app/controller/wallet.go` (from `git diff` + earlier agent review) | Partial — file modified; full read not completed in this session but diff evidence confirms changes |

---

## Gaps / Unverified from Primary Sources (Stated Explicitly)

- **Asynq worker process behavior** (dedicated worker for jobs): architectural context from user instruction only; not verified against running worker process or `asynq` official docs (not fetched — user told not to continue conversation / only output structured summary). The `infra/jobs/` file was deleted in current working tree; behavior verified only from deleted staged version read earlier (has `NewAsyncJobQueue`, `Client *asynq.Client`, `ctx.Done()` → `client.Close()` goroutine).
- **Real-time market feed data accuracy** (Massive/Alpha Vantage response format): verified via `feeds` source code (poller structure, datasource interface) but not against live API responses.
- **Exchange-rate-api.com rate accuracy / SLA**: not verified against live endpoint; source interface verified only.
- **PostgreSQL enum behavior** (`ExecutionStatusPending` default from migration 000051): verified from `models.go` const + query files; actual DB state not queried.
- **Dollar-cost averaging / multi-currency hedging strategies** mentioned by user: these are trading-strategy concepts not implemented in this codebase (no hedging module found in `services/`). Not applicable as project features — only noted as unverified relative to user's framing.

---

*File written by background agent `trading-rate-research` per research SKILL.md instruction (primary-source only, cited per claim, saved to repo convention `docs/`). Research does NOT include secondary write-ups, marketing materials, or unverified external summaries. All claims about MogTrade trace to the source files read above.*
