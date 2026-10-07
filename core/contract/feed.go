package contract

import "time"

type DatasourceQueryRequestInterval string

var (
	IvMonth    DatasourceQueryRequestInterval = "month"
	IVWeek     DatasourceQueryRequestInterval = "week"
	IVDay      DatasourceQueryRequestInterval = "day"
	IVFiveMin  DatasourceQueryRequestInterval = "5min"
	IVHour     DatasourceQueryRequestInterval = "60min"
	IVHalfHour DatasourceQueryRequestInterval = "30min"
)

type DatasourceQueryRequest struct {
	Interval DatasourceQueryRequestInterval
	Symbol   string
}
type Datasource interface {
	Pull(query DatasourceQueryRequest) ([]FeedEntry, error)
	Name() string
}

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
