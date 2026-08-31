package payloads

import (
	db "github.com/brinestone/mogtrade/internal/models"
)

type PlaceOrderPayload struct {
	Symbol           string       `json:"symbol" form:"symbol" xml:"symbol" binding:"required,min=3"`
	Side             db.OrderSide `json:"side" form:"side" xml:"side" binding:"required"`
	Type             db.OrderType `json:"type" form:"type" xml:"type" binding:"required"`
	LimitPrice       *float32     `json:"limitPrice" form:"limitPrice" xml:"limit-price"`
	StopPrice        *float32     `json:"stopPrice" form:"stopPrice" xml:"stop-price"`
	IdempotencyToken string       `header:"X-Idempotency-Token" `
}
