package payloads

type PlaceOrderPayload struct {
	Symbol           string   `json:"symbol" form:"symbol" xml:"symbol" binding:"required,min=3"`
	Side             string   `json:"side" form:"side" xml:"side" binding:"required"`
	Type             string   `json:"type" form:"type" xml:"type" binding:"required"`
	LimitPrice       *float32 `json:"limitPrice" form:"limitPrice" xml:"limit-price"`
	StopPrice        *float32 `json:"stopPrice" form:"stopPrice" xml:"stop-price"`
	IdempotencyToken string   `header:"X-Idempotency-Token" `
	Quantity         float32  `json:"quantity" form:"quantity" xml:"quantity" binding:"required,gt=3"`
}

// func (p PlaceOrderPayload) Validate() []string {
// 	msg := make([]string, 0)

// 	if p.Type == db.Limi
// OrderTypeLimit

// 	return msg
// }
