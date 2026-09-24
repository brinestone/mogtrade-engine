package eventpayloads

import "time"

type OrderPlaced struct {
	OrderId   string
	Timestamp time.Time
}
