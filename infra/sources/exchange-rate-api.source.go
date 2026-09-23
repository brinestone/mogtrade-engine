package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// exchangeRateApiSource implements ExchangeRateSource using exchangerate-api.com
type exchangeRateApiSource struct {
	apiKey string
}

func (e *exchangeRateApiSource) Name() string {
	return "exchangerate-api"
}

// PullBulk fetches exchange rates for the base currency and all target currencies in one call.
func (e *exchangeRateApiSource) PullBulk(base string, currencies []string) (map[string]float32, error) {
	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", e.apiKey, base)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read api response: %w", err)
	}

	var apiResp struct {
		Result   string             `json:"result"`
		BaseCode string             `json:"base_code"`
		Rates    map[string]float32 `json:"conversion_rates"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse api response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("api returned error result: %s", apiResp.Result)
	}

	// Filter to only the requested currencies
	result := make(map[string]float32)
	for _, currency := range currencies {
		if r, ok := apiResp.Rates[currency]; ok {
			result[currency] = r
		}
	}

	return result, nil
}

// Pull fetches a single exchange rate. Uses PullBulk internally for efficiency.
func (e *exchangeRateApiSource) Pull(base, dest string) (float32, error) {
	rates, err := e.PullBulk(base, []string{dest})
	if err != nil {
		return 0, err
	}
	if rate, ok := rates[dest]; ok {
		return rate, nil
	}
	return 0, fmt.Errorf("currency %s not available", dest)
}

// UsingExchangeRateApi creates a new ExchangeRateSource using exchangerate-api.com
func UsingExchangeRateApi(apiKey string) ExchangeRateSource {
	return &exchangeRateApiSource{
		apiKey: apiKey,
	}
}
