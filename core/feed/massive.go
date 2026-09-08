package feed

import (
	"context"
	"time"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
)

// MassiveConfig holds configuration for the Massive datasource.
type MassiveConfig struct {
	RequestTimeout time.Duration
}

// MassiveDatasource implements the feed.Datasource interface for Massive (formerly Polygon)
// using the official polygon.io client-go SDK.
type MassiveDatasource struct {
	apiKey    string
	MassiveConfig
}

// Name returns the name of the datasource.
func (ds *MassiveDatasource) Name() string {
	return "massive"
}

// Pull fetches recent aggregate bar data from Massive for the given symbol and interval.
// Uses the official polygon.io client-go SDK.
func (ds *MassiveDatasource) Pull(query DatasourceQueryRequest) (entries []FeedEntry, err error) {
	// Create the Polygon client
	client := polygon.New(ds.apiKey)

	// Calculate date range - get data from yesterday to today
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Convert the DatasourceQueryRequestInterval to polygon Timespan
	timespan := models.Timespan(string(query.Interval))

	// Create params for the ListAggs call
	params := &models.ListAggsParams{
		Ticker:    query.Symbol,
		Multiplier: 1,
		Timespan:  timespan,
		From:      models.Millis(time.UnixMilli(yesterday.UnixMilli())),
		To:          models.Millis(time.UnixMilli(now.UnixMilli())),
		// Not adjusting for splits by default to get raw prices
		Adjusted: ptr(false),
	}

	// Fetch aggregate data using the SDK
	iter := client.ListAggs(context.Background(), params)

	var aggs []models.Agg
	for iter.Next() {
		aggs = append(aggs, iter.Item())
	}
	if iter.Err() != nil {
		return nil, iter.Err()
	}

	// Convert SDK Agg models to FeedEntry
	for _, agg := range aggs {
		tradeData := MassiveTradeDataToTradeData(agg)
		// Convert Millis (time.Time) to time.Time directly
		timestamp := time.Time(agg.Timestamp)
		entry := FeedEntry{
			Source:    "Massive",
			Timestamp: timestamp,
			TradeData: tradeData,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// MassiveTradeDataToTradeData converts a polygon.io Agg model to internal TradeData.
func MassiveTradeDataToTradeData(agg models.Agg) TradeData {
	return TradeData{
		Open:   agg.Open,
		High:   agg.High,
		Low:    agg.Low,
		Close:  agg.Close,
		Volume: agg.Volume,
	}
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// NewMassiveDatasource creates a new MassiveDatasource with the given API key and configuration.
func NewMassiveDatasource(apikey string, config MassiveConfig) Datasource {
	ds := &MassiveDatasource{
		apiKey:    apikey,
		MassiveConfig: config,
	}
	return ds
}