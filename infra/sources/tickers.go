package sources

import (
	"context"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/massive-com/client-go/v3/rest"
	"github.com/massive-com/client-go/v3/rest/gen"
)

type MassiveMarketInfoConfig struct {
}

type MassiveMarketInfoProvider struct {
	apiKey string
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
