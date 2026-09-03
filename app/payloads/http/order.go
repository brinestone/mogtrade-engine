package httppayloads

import (
	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

type PlaceOrderPayload struct {
	Symbol           string              `json:"symbol" form:"symbol" xml:"symbol" binding:"required,min=3"`
	Side             string              `json:"side" form:"side" xml:"side" binding:"required"`
	Type             string              `json:"type" form:"type" xml:"type" binding:"required"`
	LimitPrice       decimal.NullDecimal `json:"limitPrice" form:"limitPrice" xml:"limit-price"`
	StopPrice        decimal.NullDecimal `json:"stopPrice" form:"stopPrice" xml:"stop-price"`
	IdempotencyToken string              `header:"X-Idempotency-Token" `
	Quantity         float32             `json:"quantity" form:"quantity" xml:"quantity" binding:"required,gt=3"`
}

func (p PlaceOrderPayload) Validate() []string {
	errors := make([]string, 0)

	if len(p.IdempotencyToken) == 0 {
		errors = append(errors, "idempotency token must be provided")
	}

	var side db.OrderSide
	if err := side.Scan(p.Side); err != nil {
		errors = append(errors, err.Error())
	}

	var ot db.OrderType
	if err := ot.Scan(p.Type); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return errors
	}

	if ot == db.OrderTypeLimit || ot == db.OrderTypeStopLimit {
		if !p.LimitPrice.Valid || p.LimitPrice.Decimal.LessThanOrEqual(decimal.Zero) {
			errors = append(errors, "Limit price required and must be positive")
		}
	}

	if ot == db.OrderTypeStop || ot == db.OrderTypeStopLimit {
		if !p.StopPrice.Valid || p.StopPrice.Decimal.LessThanOrEqual(decimal.Zero) {
			errors = append(errors, "Stop price required nad must be positive")
		}
	}
	return errors
}
