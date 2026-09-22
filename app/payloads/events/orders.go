package eventpayloads

import "time"

type OrderCreated struct {
	OrderId   string
	Timestamp time.Time
}
