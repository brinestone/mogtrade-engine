package helpers

import (
	"context"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/shopspring/decimal"
	"go-slim.dev/ioc"
)

var one = decimal.NewFromInt(1)

func GetDefaultExchangeRateFor(ctx context.Context, curr string) (decimal.Decimal, error) {
	return ioc.Call2[decimal.Decimal](ctx, func(cc contract.CurrencyConverter) (decimal.Decimal, error) {
		rates, err := cc.GetDefaultExchangeRates(ctx, curr)
		if err != nil {
			return decimal.Zero, err
		}

		rate := rates[0]
		if rate <= 0 {
			return decimal.Zero, nil
		}

		return one.Div(decimal.NewFromFloat32(rate)), nil
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
			if r <= 0 {
				rates = append(rates, decimal.Zero)
			} else {
				rates = append(rates, decimal.NewFromInt(1).Div(decimal.NewFromFloat32(r)))
			}
		}

		return rates, nil
	})
}
