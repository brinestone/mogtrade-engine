package httppayloads

import (
	"github.com/shopspring/decimal"

	"github.com/brinestone/mogtrade/core/contract"
)

type InitializeDepositRequest struct {
	Amount           decimal.Decimal `json:"amount" binding:"required"`
	Currency         string          `json:"currency" binding:"required"`
	IdempotencyToken string          `json:"client_id" binding:"required"`
	Metadata         map[string]any  `json:"metadata,omitempty"`
}

type PaymentIntentPayload struct {
	ID               string                 `json:"id"`
	ProviderRef      string                 `json:"provider_ref"`
	Provider         string                 `json:"provider"`
	Amount           decimal.Decimal        `json:"amount"`
	Currency         string                 `json:"currency"`
	Status           contract.PaymentStatus `json:"status"`
	AuthorizationURL string                 `json:"authorization_url,omitempty"`
	ClientSecret     string                 `json:"client_secret,omitempty"`
	Metadata         map[string]any         `json:"metadata,omitempty"`
}

func ToPaymentIntentPayload(p contract.PaymentIntent) PaymentIntentPayload {
	return PaymentIntentPayload{
		ID:               p.ID,
		ProviderRef:      p.ProviderRef,
		Provider:         p.Provider,
		Amount:           p.Amount,
		Currency:         p.Currency,
		Status:           p.Status,
		AuthorizationURL: p.AuthorizationURL,
		ClientSecret:     p.ClientSecret,
		Metadata:         p.Metadata,
	}
}
