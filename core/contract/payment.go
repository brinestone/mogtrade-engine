package contract

import (
	context "context"

	"github.com/shopspring/decimal"
)

// PaymentIntent represents a provider-side payment the user must complete.
type PaymentIntent struct {
	ID               string          // our internal payment id (ULID)
	ProviderRef      string          // provider's reference/access code
	Provider         string          // "paystack" | "stripe" | ...
	Amount           decimal.Decimal // Money type; never float64
	Currency         string
	Status           PaymentStatus // pending | succeeded | failed | cancelled
	AuthorizationURL string        // hosted checkout URL (optional by provider)
	ClientSecret     string        // for Stripe-style client confirmation (optional)
	Metadata         map[string]any
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
type InitializeDepositParams any

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
	Amount           decimal.Decimal
	Currency         string
	ClientID         string // our ULID / user reference
	RecipientInfo    string // bank account details, mobile money number, card token
	IdempotencyToken string // ensures retry-safe provider request
	Metadata         map[string]any
}

// PayoutResult represents the outcome of a payout attempt.
type PayoutResult struct {
	ID          string
	ProviderRef string
	Status      PaymentStatus
	Metadata    map[string]any
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
	Type        string // e.g. "payment.succeeded", "payment.failed", "payout.pending"
	PaymentID   string // our internal payment ULID
	ProviderRef string // provider's reference for the event
	Amount      decimal.Decimal
	Currency    string
	Status      PaymentStatus
	Metadata    map[string]any
}
