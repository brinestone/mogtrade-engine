package feed

import (
	"time"
)

type TradeData struct {
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type FeedEntry struct {
	TradeData
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}
