package eventpayloads

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderPlaced struct {
	OrderId   string
	Timestamp time.Time
}

type OrderCancelled struct {
	OrderId        string
	UserId         string
	Symbol         string
	RefundedAmount decimal.Decimal
	Timestamp      time.Time
}
