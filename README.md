# MogTrade Exchange Poller with Massive (formerly Polygon) Integration

## Overview

This project integrates the **Massive (formerly Polygon)** market data provider with the existing `ExchangePoller` infrastructure. The ExchangePoller's role is to poll new updates from different exchanges, and we've added support for fetching aggregate bar data from Massive.

## Architecture

### ExchangePoller
The `ExchangePoller` (in `services/market/exchange-poller.go`) polls configured data sources at regular intervals (currently every 1 second). It iterates over the provided `feed.Datasource` implementations and calls `Pull()` on each.

### Feed Datasource Interface
The `feed.Datasource` interface (in `core/feed/feed.go`) defines:
```go
type Datasource interface {
    Pull(query DatasourceQueryRequest) ([]FeedEntry, error)
}
```

### MassiveDatasource
The new `MassiveDatasource` (in `core/feed/massive.go`) implements this interface to fetch aggregate bar data from the Massive API (formerly Polygon.io).

**Key features:**
- Fetches recent aggregate bars for a given symbol and time interval
- Converts Massive's response format to the internal `TradeData` structure
- Implements the `feed.Datasource` interface seamlessly
- Configurable request timeout via `MassiveConfig`

## Massive API Integration

The MassiveDatasource queries the Massive API endpoint:
```
https://api.polygon.io/v2/aggs/ticker/{symbol}/range/{multiplier}/{timespan}/{from}/{to}?apiKey={apikey}
```

**Supported parameters:**
- `symbol`: Trading symbol (e.g., "AAPL")
- `interval`: Time range interval (5min, 30min, 60min, daily, weekly, monthly)
- `from`/`to`: Date range in YYYY-MM-DD format

**Response fields converted to TradeData:**
- Open, High, Low, Close prices
- Volume

## Configuration

### Environment Variables

The following environment variable is required (already present in `app/cmd/.env`):

```
MASSIVE_API_KEY=k2ZDg6YupZ45k7Wh8DtMpvplzwvm1cuD
```

### IOC Configuration

The integration is configured in `app/cmd/ioc.go`:

```go
if err := ioc.Factory(func(h *market.ExchangeHub, l *slog.Logger) *market.ExchangePoller {
    return market.NewExchangePoller(l.With("service", "exchange-poller"), h, []feed.Datasource{
        feed.NewMassiveDatasource(os.Getenv("MASSIVE_API_KEY"), feed.MassiveConfig{RequestTimeout: 10 * time.Second}),
    })
}, true); err != nil {
    return err
}
```

## Building and Running

The project builds successfully with:

```bash
cd I:/mogtrade-engine/app && go build ./...
```

## What Was Done

1. **Created `core/feed/massive.go`** - New MassiveDatasource implementing the feed.Datasource interface
2. **Updated `app/cmd/ioc.go`** - Integrated MassiveDatasource into the ExchangePoller via DI container
3. **Verified compilation** - All Go code compiles without errors
4. **Leveraged existing infrastructure** - Uses the same `.env` pattern as AlphaVantage and other APIs

## Next Steps

- Implement symbol scheduling/management for the Massive datasource
- Add proper error handling and rate limiting
- Support additional Massive API endpoints (crypto, futures, etc.)
- Integrate with the ExchangeHub broadcasting mechanism