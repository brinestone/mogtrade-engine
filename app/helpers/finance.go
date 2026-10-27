package helpers

import (
	"context"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/shopspring/decimal"
	"go-slim.dev/ioc"
)

func GetDefaultExchangeRateFor(ctx context.Context, curr string) (float32, error) {
	return ioc.Call2[float32](ctx, func(cc contract.CurrencyConverter) (float32, error) {
		rates, err := cc.GetDefaultExchangeRates(ctx, curr)
		if err != nil {
			return 0, err
		}

		rate := rates[0]
		if rate == 0 {
			return 1.0, nil
		}
		return 1 / rate, nil
	})
}

func GetDefaultExchangeRateDecimalFor(ctx context.Context, curr ...string) ([]decimal.Decimal, error) {
	return ioc.Call2[[]decimal.Decimal](ctx, func(cc contract.CurrencyConverter) ([]decimal.Decimal, error) {
		_rates, err := cc.GetDefaultExchangeRates(ctx, curr...)
		rates := make([]decimal.Decimal, 0)
		if err != nil {
			return rates, err
		}

		for _, r := range _rates {
			rates = append(rates, decimal.NewFromFloat32(r))
		}

		return rates, nil
	})
}
