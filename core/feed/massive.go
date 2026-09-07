package feed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MassiveConfig holds configuration for the Massive datasource.
type MassiveConfig struct {
	RequestTimeout time.Duration
}

// MassiveTradeData represents the trade data returned from Massive API.
type MassiveTradeData struct {
	Open   float64 `json:"o"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Close  float64 `json:"c"`
	Volume int64   `json:"v"`
}

// MassiveAggregate represents an aggregate bar from Massive API.
type MassiveAggregate struct {
	Timestamp int64            `json:"t"`
	Open      float64          `json:"o"`
	High      float64          `json:"h"`
	Low       float64          `json:"l"`
	Close     float64          `json:"c"`
	Volume    int64            `json:"v"`
	WeightedAvg float64        `json:"w"`
	Transactions int64        `json:"n"`
	VWAP      float64         `json:"vw"`
}

// MassiveResponse represents the API response structure.
type MassiveResponse struct {
	Status      string             `json:"status"`
	Results     []MassiveAggregate `json:"results"`
	Error       string             `json:"error"`
	ErrorCode   string             `json:"error_code"`
}

// MassiveDatasource implements the feed.Datasource interface for Massive (formerly Polygon).
type MassiveDatasource struct {
	apiKey    string
	MassiveConfig
}

// Pull fetches recent aggregate data from Massive for the given symbol and interval.
func (ds *MassiveDatasource) Pull(query DatasourceQueryRequest) (entries []FeedEntry, err error) {
	client := ds.newClient()

	// Massive API endpoint for aggregates
	// Format: https://api.polygon.io/v2/aggs/ticker/{symbol}/range/{multiplier}/{timespan}/{from}/{to}?apiKey={apikey}
	url := fmt.Sprintf("https://api.polygon.io/v2/aggs/ticker/%s/range/%d/%s/%s/%s?apiKey=%s",
		query.Symbol,
		1, // multiplier
		string(query.Interval),
		formatDate(time.Now().Add(-24*time.Hour)), // yesterday
		formatDate(time.Now()),                    // today
		ds.apiKey,
	)

	res, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to request Massive data: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("unexpected status from Massive: %d - %s", res.StatusCode, string(body))
	}

	jsonBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Massive response: %w", err)
	}

	var response MassiveResponse
	if err = json.Unmarshal(jsonBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse Massive response: %w", err)
	}

	if response.Status != "OK" {
		return nil, fmt.Errorf("Massive API error: %s", response.Error)
	}

	// Convert Massive aggregates to FeedEntry
	for _, agg := range response.Results {
		tradeData := MassiveTradeDataToTradeData(agg)
		entry := FeedEntry{
			Source:    "Massive",
			Timestamp: time.Unix(agg.Timestamp/1000, 0), // Massive uses milliseconds
			TradeData: tradeData,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// MassiveTradeDataToTradeData converts a MassiveAggregate to TradeData.
func MassiveTradeDataToTradeData(agg MassiveAggregate) TradeData {
	return TradeData{
		Open:   agg.Open,
		High:   agg.High,
		Low:    agg.Low,
		Close:  agg.Close,
		Volume: float64(agg.Volume),
	}
}

// newClient creates a new HTTP client with the configured timeout.
func (ds *MassiveDatasource) newClient() *http.Client {
	return &http.Client{Timeout: ds.RequestTimeout}
}

// formatDate formats a time.Time to YYYY-MM-DD string for Massive API.
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// NewMassiveDatasource creates a new MassiveDatasource with the given API key and configuration.
func NewMassiveDatasource(apikey string, config MassiveConfig) Datasource {
	ds := &MassiveDatasource{
		apiKey:    apikey,
		MassiveConfig: config,
	}
	return ds
}