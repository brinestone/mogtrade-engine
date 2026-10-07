package contract

import "context"

type NullTickerInfo struct {
	TickerInfo
	Valid bool
}

type TickerInfo struct {
	Symbol   string `json:"symbol"`
	Currency string `json:"currency"`
	Exchange string `json:"exchange"`
	Name     string `json:"name"`
	Locale   string `json:"locale"`
	Market   string `json:"market"`
}

type ExchangeInfo struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Acronym string `json:"acronym"`
	Symbol  string `json:"symbol"`
}

type FindTickerInfoParams struct {
	Limit int32
	Query *string
}
type FindExchangeInfoParams struct {
	Limit int32
	Query *string
}
type TickerInfoProvider interface {
	FindTickers(context.Context, FindTickerInfoParams) ([]TickerInfo, error)
	FindTickerInfoBySymbol(context.Context, string) (NullTickerInfo, error)
}
type ExchangeInfoProvider interface {
	FindExchanges(context.Context, FindExchangeInfoParams) ([]ExchangeInfo, error)
}
type CurrencyConverter interface {
	GetDefaultExchangeRates(context.Context, ...string) ([]float32, error)
	GetExchangeRates(context.Context, string, ...string) ([]float32, error)
}
