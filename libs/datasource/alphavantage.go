package datasource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type AVTimeRange string
type AVFunction string

var (
	AVFIntraDay AVFunction = "TIME_SERIES_INTRADAY"
	AVFDaily    AVFunction = "TIME_SERIES_DAILY"
	AVFWeekly   AVFunction = "TIME_SERIES_WEEKLY"
	AVFMonthly  AVFunction = "TIME_SERIES_MONTHLY"
)

var (
	AVTOneMinute      AVTimeRange = "1min"
	AVTFiveMinutes    AVTimeRange = "5min"
	AVTFifteenMinutes AVTimeRange = "15min"
	AVTHour           AVTimeRange = "60min"
	AVTHalfHour       AVTimeRange = "30min"
)

type AlphaVantageConfig struct {
	RequestTimeout time.Duration
}

type AlphaVantageDatasource struct {
	apiKey string
	AlphaVantageConfig
}

func (ds *AlphaVantageDatasource) newClient() *http.Client {
	return &http.Client{Timeout: ds.RequestTimeout}
}

func (req DatasourceQueryRequestInterval) toAlphaVantageFunction() AVFunction {
	switch req {
	case IVDay:
		return AVFDaily
	case IVWeek:
		return AVFWeekly
	case IvMonth:
		return AVFMonthly
	default:
		return AVFIntraDay
	}
}

func (req DatasourceQueryRequestInterval) toAlphaVantageInterval() string {
	switch req {
	case IVHour:
		return "60min"
	case IVHalfHour:
		return "30min"
	default:
		return "5min"
	}
}

func (req DatasourceQueryRequest) toAlphaVantageQueryUrl(apiKey string) string {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=%s&symbol=%s&interval=%s&apikey=%s", req.Interval.toAlphaVantageFunction(), req.Symbol, req.Interval.toAlphaVantageInterval(), apiKey)
	return url
}

func (ds *AlphaVantageDatasource) Pull(query DatasourceQueryRequest) (entries []FeedEntry, err error) {
	client := ds.newClient()

	res, err := client.Get(query.toAlphaVantageQueryUrl(ds.apiKey))
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected status: %s\n", res.Status)
		return
	}

	jsonBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	var result map[string]any
	if err = json.Unmarshal(jsonBytes, &result); err != nil {
		return
	}

	var key string
	var timeFormat string
	switch query.Interval.toAlphaVantageFunction() {
	case AVFDaily:
		key = "Time Series (Daily)"
	case AVFMonthly:
		key = "Monthly Time Series"
	case AVFWeekly:
		key = "Weekly Time Series"
	case AVFIntraDay:
		key = fmt.Sprintf("Time Series (%s)", query.Interval)
	}

	switch query.Interval.toAlphaVantageFunction() {
	case AVFDaily:
	case AVFWeekly:
	case AVFMonthly:
		timeFormat = time.DateOnly
	case AVFIntraDay:
		timeFormat = "2006-01-02 15:04:05"
	}

	timeSeries := result[key].(map[string]any)

	for key, value := range timeSeries {
		timestamp, err := time.Parse(timeFormat, key)
		if err != nil {
			break
		}

		tradeData, err := mapToAlphaVantageTradeData(value.(map[string]any))
		if err != nil {
			break
		}
		entry := FeedEntry{
			Source:    "AlphaVantage",
			Timestamp: timestamp,
			TradeData: tradeData,
		}
		entries = append(entries, entry)
	}
	if err != nil {
		entries = make([]FeedEntry, 0)
	}

	return
}

func mapToAlphaVantageTradeData(m map[string]any) (TradeData, error) {
	data := TradeData{}
	high, err := strconv.ParseFloat(m["2. high"].(string), 64)
	if err != nil {
		return data, err
	}
	open, err := strconv.ParseFloat(m["1. open"].(string), 64)
	if err != nil {
		return data, err
	}
	low, err := strconv.ParseFloat(m["3. low"].(string), 64)
	if err != nil {
		return data, err
	}
	close, err := strconv.ParseFloat(m["4. close"].(string), 64)
	if err != nil {
		return data, err
	}
	volume, err := strconv.ParseFloat(m["5. volume"].(string), 64)
	if err != nil {
		return data, err
	}

	data.High = high
	data.Open = open
	data.Low = low
	data.Close = close
	data.Volume = volume

	return data, nil
}

func NewAlphaVantageDatasource(apikey string, config AlphaVantageConfig) Datasource {
	ds := &AlphaVantageDatasource{
		apiKey:             apikey,
		AlphaVantageConfig: config,
	}
	return ds
}
