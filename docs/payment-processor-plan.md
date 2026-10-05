# Payment Processor Integration Plan for MogTrade

## Overview

This plan defines how MogTrade integrates external payment processors (Stripe, Paystack, Flutterwave, etc.) to fund user wallets and process withdrawals. The key distinction: **Payment Transactions** (external processor events) and **Wallet Transactions** (internal ledger entries) are separate concepts that only interact when funds enter the system.

## 1. Core Distinction: Payment vs. Wallet Transactions

### 1.1 Payment Transactions (External)
- **Origin**: External payment processor (Stripe, Paystack, Flutterwave, Adyen)
- **Purpose**: Receive funds from customers, process payouts to users, handle refunds/disputes
- **Lifecycle**: Initiated by customer → processed by provider → webhook-verified → settled
- **ID**: Provider reference + our internal payment ULID
- **Storage**: `payments` table in DB
- **Examples**: Customer depositing money, user withdrawing funds, refund after dispute

### 1.2 Wallet Transactions (Internal)
- **Origin**: MogTrade's internal ledger
- **Purpose**: Track wallet balance changes, record deposits/withdrawals, maintain audit trail
- **Lifecycle**: Created when payment succeeds → links to payment record → updates wallet snapshot
- **ID**: Our internal ULID
- **Storage**: `wallet_transactions` table in DB
- **Examples**: Depositing received funds into user wallet, withdrawing funds out of user wallet, adjusting balance on dispute

### 1.3 The Interaction Point
The only interaction occurs when:
1. A **Payment Transaction** succeeds (webhook verified)
2. A **Wallet Transaction** is created to credit the user's wallet
3. The `wallet_transaction_id` on the `payments` table links back

**Never** create a wallet transaction before provider confirmation. **Never** mark a payment succeeded before creating the associated wallet transaction (for deposits).

## 2. Revised Contract Design (`core/contract/payment.go`)

```go
package contract

// PaymentIntent represents a provider-side payment the user must complete.
type PaymentIntent struct {
    ID            string          // our internal payment id (ULID)
    ProviderRef   string          // provider's reference/access code
    Provider      string          // "paystack" | "stripe" | ...
    Amount        decimal.Decimal // Money type; never float64
    Currency      string
    Status        PaymentStatus // pending | succeeded | failed | cancelled
    AuthorizationURL string       // hosted checkout URL (optional by provider)
    ClientSecret  string          // for Stripe-style client confirmation (optional)
    Metadata      map[string]any
}

// PaymentStatus represents the lifecycle state of a payment.
type PaymentStatus string

const (
    PaymentStatusPending   PaymentStatus = "pending"
    PaymentStatusSucceeded PaymentStatus = "succeeded"
    PaymentStatusFailed    PaymentStatus = "failed"
    PaymentStatusCancelled PaymentStatus = "cancelled"
)

// DepositProcessor initiates and queries inbound payments (wallet funding).
type DepositProcessor interface {
    // InitializeDeposit creates a provider payment and returns a reference/URL
    // the client uses to complete payment.
    InitializeDeposit(ctx context.Context, p InitializeDepositParams) (PaymentIntent, error)
    // VerifyDeposit re-queries the provider for the authoritative status of a reference.
    VerifyDeposit(ctx context.Context, providerRef string) (PaymentIntent, error)
}

// InitializeDepositParams contains the inputs for creating a deposit payment.
type InitializeDepositParams struct {
    Amount        decimal.Decimal
    Currency      string
    ClientID      string          // our user ID / ULID
    IDempotencyToken string        // ensures retry-safe provider request
    Metadata      map[string]any  // e.g. order reference, user prefs
}

// PayoutProcessor sends money out (withdrawals). Split from deposits so
// providers that only support one direction still satisfy the contract.
type PayoutProcessor interface {
    // CreatePayout initiates a provider payout (withdrawal to user's bank/card/mobile money).
    CreatePayout(ctx context.Context, p CreatePayoutParams) (PayoutResult, error)
    // VerifyPayout re-queries the provider for the authoritative status of a payout.
    VerifyPayout(ctx context.Context, providerRef string) (PayoutResult, error)
}

// CreatePayoutParams contains the inputs for creating a payout.
type CreatePayoutParams struct {
    Amount        decimal.Decimal
    Currency      string
    ClientID      string          // our ULID / user reference
    RecipientInfo string          // bank account details, mobile money number, card token
    IDempotencyToken string        // ensures retry-safe provider request
    Metadata      map[string]any
}

// PayoutResult represents the outcome of a payout attempt.
type PayoutResult struct {
    ID            string
    ProviderRef   string
    Status        PaymentStatus
    Metadata      map[string]any
}

// WebhookHandler validates and normalizes provider webhook payloads.
type WebhookHandler interface {
    // VerifySignature validates the raw body against the provider signature header.
    VerifySignature(rawBody []byte, signatureHeader string) bool
    // ParseEvent normalizes the provider event into an internal PaymentEvent.
    ParseEvent(rawBody []byte) (PaymentEvent, error)
}

// PaymentEvent is the normalized internal representation of a provider webhook event.
type PaymentEvent struct {
    Type       string              // e.g. "payment.succeeded", "payment.failed", "payout.pending"
    PaymentID  string              // our internal payment ULID
    ProviderRef string             // provider's reference for the event
    Amount     decimal.Decimal
    Currency   string
    Status     PaymentStatus
    Metadata   map[string]any
}
```

## 3. Data Model (sqlc + migrations)

### 3.1 Payments Table (NEW)

```sql
-- +goose Up
CREATE TYPE payment_direction AS ENUM ('deposit', 'payout');
CREATE TYPE payment_status AS ENUM ('pending', 'succeeded', 'failed', 'cancelled');

CREATE TABLE payments (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES "user"("id") ON DELETE SET NULL,
    provider TEXT NOT NULL,               -- free-text; phase 8 adds enum if desired
    direction payment_direction NOT NULL,
    provider_ref TEXT NOT NULL,          -- provider's reference / access code
    idempotency_token TEXT NOT NULL UNIQUE,
    amount NUMERIC(18,4) NOT NULL,
    currency TEXT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    wallet_transaction_id VARCHAR(26) REFERENCES wallet_transactions(id), -- FK, nullable
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_ref)
);
CREATE INDEX idx_payments_user ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);

-- +goose Down
DROP TABLE IF EXISTS payments;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_direction;
```

**Rationale**:
- `wallet_transaction_id` is **nullable** (FK optional) because withdrawals may not yet have a wallet transaction record, and historical payments won't have one.
- `NUMERIC(18,4)` matches every other money column in the schema.
- `provider TEXT` (not enum) keeps the DB layer agnostic; migration can add enum later.
- `idempotency_token TEXT` matches the existing `wallet_transactions.idempotency_token` column type.

### 3.2 Wallet Transactions (EXISTING - no changes needed)

The existing `wallet_transactions` table and `RecordWalletTransaction`/`UpdateWalletTransactionStatus` functions remain unchanged. They are only created **after** a deposit payment succeeds.

## 4. Flows

### 4.1 Deposit Flow (Hosted-Checkout Pattern, e.g. Paystack)

```mermaid
flowchart TD
    %% Client Side
    Client -->|POST /api/v1/payments/deposits {amount, currency}| Controller
    
    %% Controller Phase
    Controller -->|Generate ULID, idempotency token| DB[(payments table)]
    DB -->|Insert row (status=pending)| Controller
    Controller -->|Call contract.DepositProcessor.InitializeDeposit| Provider[External Provider]
    Provider -->|Return authorizationUrl + reference| Controller
    Controller -->|Return {authorizationUrl, reference}| Client
    
    %% User Side
    Client -->|User completes payment on provider page| Provider
    Provider -->|Payment processed| Provider
    
    %% Webhook Phase
    Provider -->|POST /api/v1/payments/webhooks/{provider} (raw body)| Webhook Handler
    Webhook Handler -->|Verify signature (HMAC)| Webhook Handler
    Webhook Handler -->|Look up payment (provider, provider_ref)| DB
    DB -->|Ignore if already succeeded (idempotent)| Webhook Handler
    
    %% Success Path
    Webhook Handler -->|In one DB transaction:
        UPDATE payments SET status='succeeded' WHERE id = $1
        RecordWalletTransaction(deposit) -- Src: systemWallet, Dest: userWallet, Intent: "deposit"
        UPDATEWalletTransactionStatus created→processing→completed
        Publish payment.deposit.completed.v1 on EventBus
    |--> Webhook Handler
    
    %% Failure Path
    Webhook Handler -->|On failure: UPDATE payments SET status='failed'| Webhook Handler
    Webhook Handler -->|Optional: RecordWalletTransaction with Intent: "deposit reversal"| Webhook Handler
    
    %% Client Polling (optional)
    Client -->|GET /api/v1/payments/deposits/{id}| Controller
    Controller -->|Call VerifyDeposit (only if still pending)| Provider
    Provider -->|Return current status| Controller
    Controller -->|Return status in JSON| Client
```

### 4.2 Withdrawal Flow

```mermaid
flowchart TD
    %% Service Phase
    Service -->|Validate balance| DB[(wallet + payments)]
    DB -->|Debit wallet (created→processing→completed)| WalletLedger[wallet_transactions]
    DB -->|Create payment (dir=payout, status=pending)| Payments[payments table]
    DB -->|Call PayoutProcessor.CreatePayout| Provider[External Provider]
    Provider -->|Return provider reference| Service
    
    %% Webhook Phase
    Provider -->|POST /api/v1/payments/webhooks/{provider} (raw body)| Webhook Handler
    Webhook Handler -->|Verify signature| Webhook Handler
    Webhook Handler -->|Look up payment (provider, provider_ref)| DB
    DB -->|If succeeded:
        UPDATE payments SET status='succeeded'
        UPDATEWalletTransactionStatus processing→failed (or completed)
    |--> Webhook Handler
    
    %% Refund on Failure
    Webhook Handler -->|If failed:
        UPDATEWalletTransactionStatus processing→failed
        UPDATE payments SET status='failed'
        Do NOT post reversal entry (wallet_snapshots excludes failed)
    |--> Webhook Handler
```

### 4.3 Refund Flow (Payout Failure)

```mermaid
flowchart TD
    Webhook -->|VerifyPayout returns failed| Service
    Service -->|In same DB tx:
        UPDATEWalletTransactionStatus processing→failed
        UPDATE payments SET status='failed'
    |--> Service
    Service -->|Do NOT post second reversal entry| Service
    Service -->|wallet_snapshots view excludes failed transactions| Service
    Service -->|Publish payment.payout.failed.v1 on EventBus| Service
```

## 5. IOC Wiring (app module)

```go
ioc.Factory(func(logger *slog.Logger, cfg PaymentConfig) contract.DepositProcessor { ... }, true)
// provider selected via PAYMENT_PROVIDER env var, mirroring STORAGE_PROVIDER pattern

ioc.Factory(func(logger *slog.Logger, cfg PaymentConfig) contract.PayoutProcessor { ... }, true)
// provider selected via PAYOUT_PROVIDER env var
```

## 5. Error Handling & Reliability

| Failure Mode | Automatic Handling | Manual Admin Action |
|---|---|---|
| **Provider API down** | Deposits stay `pending`; retry job (`VerifyDeposit`) reconciles stale entries | Admin monitors provider status; may switch fallback provider |
| **Duplicate webhooks** | Unique `(provider, provider_ref)` + status guard → idempotent processing | None needed |
| **Signature failure** | 400 response; log via `slog`; never 500 (avoids provider retry storms) | Admin rotates webhook secret if compromised |
| **Replay protection** | Per-provider timestamp tolerance (`Stripe-Signature` 10-min); Paystack has no tolerance | Document per-provider behavior in `infra/payments/*` |
| **Double-crediting wallet** | Unique idempotency token + unique provider ref + DB transaction around credit | None needed (code-enforced) |

## 6. Phased Implementation

| Phase | Scope |
|-------|-------|
| 1 | Redesign `core/contract/payment.go` (types + interfaces above) — no behavior change |
| 2 | Migrations + sqlc queries for `payments` table (names and DDL as specified) |
| 3 | `services/billing/deposit.go` + `services/billing/withdrawal.go` business logic |
| 3.5 | **Key**: Ensure `RecordWalletTransaction` only called AFTER payment webhook verifies success |
| 4 | `infra/payments/paystack.go` first concrete provider (sandbox) |
| 5 | Controllers + routes: init deposit, status, webhook (public, raw body read before binding) |
| 6 | Withdrawal flow + payout support |
| 7 | Reconciliation job for stale pending payments (also sweep stalled payouts) |
| 8 | Stripe as second provider to pressure-test the contract abstraction |

## 7. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| **Contract leaks provider details** | Review rule: `core/contract` must compile with zero provider imports; returning `AuthorizationURL` AND `ClientSecret` optional fields covers both Stripe and Paystack patterns without naming providers |
| **Webhook before user redirect completes** | Webhook is authoritative; status endpoint reconciles |
| **Double-crediting wallet** | Unique idempotency token + unique provider ref + DB transaction around credit |
| **Provider API drift** | Contract-only dependency isolates the change to `infra/payments/*` |
| **Money as float64 in existing stub** | Replace all payment amounts with `decimal.Decimal` (project rule already used in billing) |
| **Secrets in code** | Provider keys via `.env` (gitignored), `STRIPE_SECRET_KEY`, `PAYSTACK_SECRET_KEY`, webhook secrets per provider |
| **Webhook signature replay / race conditions** | Per-provider `VerifySignature` implementations with documented replay windows; reconcile job uses row-level locks (`SELECT ... FOR UPDATE`) to avoid concurrent webhook processing |

## 7. Appendices

### Appendix A: Provider-Specific Webhook Header Mapping
- **Stripe**: header `Stripe-Signature` → parse `t=...,v1=...,v0=...`; verify with `endpointSecret` and 10-min timestamp tolerance.
- **Paystack**: header `x-paystack-signature` → HMAC-SHA512 of raw body using the secret key; no timestamp (replay not an issue but also no tolerance).
- **Flutterwave**: header `Flutterwave-Signature` → HMAC-SHA256 of raw body using the secret key.

### Appendix B: Reconciliation Job (pseudo-code)
```go
rows, _ := db.Queries.GetStalePayments(ctx, time.Now().Add(-48*time.Hour))
for _, p := range rows {
    // row-level lock to avoid racing the webhook
    tx, _ := db.Pool.Begin(ctx)
    updated, _ := db.Queries.VerifyPayment(ctx, tx, p.ID)
    if updated {
        // complete the flow
        db.Queries.Commit(tx)
    } else {
        db.Queries.Rollback(tx)
    }
}
```

### Appendix C: Glossary
- **ULID**: Universally Unique Lexicographically Sortable Identifier — project standard for IDs.
- **decimal.Decimal**: Go type from `shopspring/decimal` — project standard for Money; never `float64`.
- **Idempotency Token**: client-supplied string guaranteeing retry-safe API calls; maps to `wallet_transactions.idempotency_token` DB column.
- **Event Bus**: `contract.EventBus` — `Subscribe`/`Publish` channels for domain events (`order.created.v1`, `payment.deposit.completed.v1`, etc.).

### Appendix C: Interaction Summary Table

| Scenario | Payment Transaction Created | Wallet Transaction Created | Link |
|---|---|---|---|
| **Deposit succeeds** | Yes (status→succeeded) | Yes (Src: systemWallet, Dest: userWallet, Intent: "deposit") | `wallet_transaction_id` FK on payments |
| **Withdrawal succeeds** | Yes (status→succeeded) | Yes (debit wallet, Status: processing→completed) | `wallet_transaction_id` FK on payments |
| **Deposit fails** | Yes (status→failed) | No (or reversal entry, but wallet_snapshots excludes failed) | FK nullable; optional reversal |
| **Withdrawal fails** | Yes (status→failed) | Yes (Status: processing→failed) | FK nullable; no reversal entry (avoids double-count) |
| **Historical payment** (pre-plan) | Yes | No (plan didn't exist) | FK is NULL |

---
*Plan version 1.0 — generated 2024. Based on MogTrade codebase context: existing wallet transaction logic in `services/billing/wallet.go`, existing `payments` table concept, existing `RecordWalletTransaction` function, existing `wallet_snapshots` view.*