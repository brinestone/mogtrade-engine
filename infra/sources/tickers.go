package sources

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/massive-com/client-go/v3/rest"
	"github.com/massive-com/client-go/v3/rest/gen"
)

type MassiveMarketInfoConfig struct {
}

type MassiveMarketInfoProvider struct {
	apiKey string
}

// FindTickerInfoBySymbol implements [contract.TickerInfoProvider].
func (m *MassiveMarketInfoProvider) FindTickerInfoBySymbol(ctx context.Context, symbol string) (contract.NullTickerInfo, error) {
	client := rest.NewWithOptions(m.apiKey)
	params := &gen.GetTickerParams{}
	res, err := client.GetTicker(ctx, symbol, params)
	if err != nil {
		return contract.NullTickerInfo{}, err
	}
	err = rest.CheckResponse(res)
	if err != nil {
		return contract.NullTickerInfo{}, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return contract.NullTickerInfo{}, err
	}
	var payload getTickerResponse
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return contract.NullTickerInfo{}, err
	}

	return contract.NullTickerInfo{
		Valid: true,
		TickerInfo: contract.TickerInfo{
			Symbol:   symbol,
			Currency: payload.Results.CurrencyName,
			Exchange: payload.Results.PrimaryExchange,
			Name:     payload.Results.Name,
			Locale:   payload.Results.Locale,
			Market:   payload.Results.Market,
		},
	}, nil
}

func NewMassiveMarketInfoProvider(apiKey string) *MassiveMarketInfoProvider {
	return &MassiveMarketInfoProvider{
		apiKey: apiKey,
	}
}

func (m *MassiveMarketInfoProvider) FindExchanges(ctx context.Context, p contract.FindExchangeInfoParams) ([]contract.ExchangeInfo, error) {
	result := make([]contract.ExchangeInfo, 0)

	client := rest.NewWithOptions(m.apiKey, rest.WithPagination(true))
	params := &gen.ListExchangesParams{}
	res, err := client.ListExchangesWithResponse(ctx, params)
	if err != nil {
		return result, err
	}

	iter := rest.NewIteratorFromResponse(client, res)
	if iter.Err() != nil {
		return result, nil
	}
	for iter.Next() {
		item := iter.Item()
		result = append(result, contract.ExchangeInfo{
			Symbol:  item["ticker"].(string),
			Name:    item["name"].(string),
			Type:    item["type"].(string),
			Acronym: item["acronym"].(string),
		})
	}
	return result, nil
}

func (m *MassiveMarketInfoProvider) FindTickers(ctx context.Context, p contract.FindTickerInfoParams) ([]contract.TickerInfo, error) {
	result := make([]contract.TickerInfo, 0)

	client := rest.NewWithOptions(m.apiKey, rest.WithPagination(true))
	params := &gen.ListTickersParams{
		Order: new(gen.ListTickersParamsOrderAsc),
		Limit: new(int(p.Limit)),
	}
	res, err := client.ListTickersWithResponse(ctx, params)
	if err != nil {
		return result, err
	}

	iter := rest.NewIteratorFromResponse(client, res)
	if iter.Err() != nil {
		return result, nil
	}
	for iter.Next() {
		item := iter.Item()
		result = append(result, contract.TickerInfo{
			Symbol:   item["ticker"].(string),
			Currency: item["currency_name"].(string),
			Exchange: item["primary_exchan"].(string),
			Name:     item["name"].(string),
			Locale:   item["locale"].(string),
			Market:   item["market"].(string),
		})
	}
	return result, nil
}

type getTickerResponse struct {
	RequestId string  `json:"request_id"`
	Results   results `json:"results"`
	Status    string  `json:"status"`
}

type results struct {
	Ticker                      string    `json:"ticker"`
	Name                        string    `json:"name"`
	Market                      string    `json:"market"`
	Locale                      string    `json:"locale"`
	PrimaryExchange             string    `json:"primary_exchange"`
	Type                        string    `json:"type"`
	Active                      bool      `json:"active"`
	CurrencyName                string    `json:"currency_name"`
	Cik                         string    `json:"cik"`
	CompositeFigi               string    `json:"composite_figi"`
	ShareClassFigi              string    `json:"share_class_figi"`
	MarketCap                   float64   `json:"market_cap"`
	PhoneNumber                 string    `json:"phone_number"`
	Address                     address   `json:"address"`
	Description                 string    `json:"description"`
	SicCode                     string    `json:"sic_code"`
	SicDescription              string    `json:"sic_description"`
	TickerRoot                  string    `json:"ticker_root"`
	HomepageUrl                 string    `json:"homepage_url"`
	TotalEmployees              float64   `json:"total_employees"`
	ListDate                    time.Time `json:"list_date"`
	Branding                    Branding  `json:"branding"`
	ShareClassSharesOutstanding float64   `json:"share_class_shares_outstanding"`
	WeightedSharesOutstanding   float64   `json:"weighted_shares_outstanding"`
	RoundLot                    float64   `json:"round_lot"`
}

type address struct {
	Address1   string `json:"address1"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

type Branding struct {
	LogoUrl string `json:"logo_url"`
	IconUrl string `json:"icon_url"`
}
