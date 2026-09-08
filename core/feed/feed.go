package feed

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
