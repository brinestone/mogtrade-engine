package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
	"log"
)

// cacheEntry represents a single entry in the file-backed cache.
type cacheEntry struct {
	Rates    map[string]float32 // symbol -> rate (against default base)
	Timestamp int64              // Unix seconds when cached
}

// CachedCurrencyConverter implements the CurrencyConverter interface using
// a file-backed cache for exchange rates. On cache-miss, the converter
// fetches rates from the API and stores them in the cache with a
// configurable TTL.
type CachedCurrencyConverter struct {
	mu          sync.RWMutex
	cacheFile   *os.File
	cachePath   string // path to cache file
	ttlSeconds  int64
	defaultBase string
	apiBaseURL  string
	apiKey      string // API key for the exchangerate service
	logger      *log.Logger
}

// NewCachedCurrencyConverter creates a new CachedCurrencyConverter.
func NewCachedCurrencyConverter(ttlSeconds int64, defaultBase string, apiBaseURL string, apiKey string) *CachedCurrencyConverter {
	cc := &CachedCurrencyConverter{
		ttlSeconds:  ttlSeconds,
		defaultBase: defaultBase,
		apiBaseURL:  apiBaseURL,
		apiKey:      apiKey,
		logger:      log.New(os.Stderr, "[currency-converter] ", log.LstdFlags),
	}
	cc.initializeCache()
	return cc
}

// initializeCache opens/creates the cache file.
func (cc *CachedCurrencyConverter) initializeCache() {
	var err error

	// Create cache file in temp directory, or use a fixed path
	cc.cachePath = "/tmp/mogtrade-currency-cache.json"

	// Ensure directory exists
	if err := os.MkdirAll("/tmp", 0755); err != nil {
		// fallback to current directory
		cc.cachePath = "currency-cache.json"
	}

	// Create/truncate the cache file
	cc.cacheFile, err = os.Create(cc.cachePath)
	if err != nil {
		// fallback
		cc.cacheFile, err = os.Create(cc.cachePath)
		if err != nil {
			cc.cacheFile = nil
		}
	}
}

// closeCache closes the cache file.
func (cc *CachedCurrencyConverter) closeCache() {
	if cc.cacheFile != nil {
		cc.cacheFile.Close()
	}
}

// lookupCache checks the file-backed cache for rates against the default base.
// Returns the rates map and whether the cache entry is valid (not expired).
func (cc *CachedCurrencyConverter) lookupCache() (map[string]float32, bool) {
	if cc.cacheFile == nil {
		cc.logger.Println("lookupCache: cache file not available, returning cache miss")
		return nil, false
	}

	data, err := os.ReadFile(cc.cachePath)
	if err != nil {
		cc.logger.Printf("lookupCache: failed to read cache file: %v, returning cache miss", err)
		return nil, false
	}

	var rates map[string]float32
	if err := json.Unmarshal(data, &rates); err != nil {
		cc.logger.Printf("lookupCache: failed to unmarshal cache JSON: %v, returning cache miss", err)
		return nil, false
	}

	// Check TTL: verify the cache was written recently
	now := time.Now().Unix()
	fileInfo, _ := cc.cacheFile.Stat()
	fileModTime := fileInfo.ModTime().Unix()

	// If cache file was modified within TTL, consider it valid
	if now-fileModTime <= cc.ttlSeconds && len(rates) > 0 {
		cc.logger.Printf("lookupCache: cache hit (TTL: %d seconds remaining, %d rates)", now-fileModTime, len(rates))
		return rates, true
	}

	cc.logger.Printf("lookupCache: cache miss (expired or empty, TTL check: %d seconds old)", now-fileModTime)
	return nil, false
}

// populateCache writes the given rates to the cache file.
func (cc *CachedCurrencyConverter) populateCache(rates map[string]float32) {
	data, err := json.Marshal(rates)
	if err != nil {
		cc.logger.Printf("populateCache: failed to marshal rates to JSON: %v", err)
		return
	}

	// Write to cache file
	if err := os.WriteFile(cc.cachePath, data, 0644); err != nil {
		cc.logger.Printf("populateCache: failed to write cache file: %v", err)
		return
	}
	cc.logger.Println("populateCache: successfully wrote rates to cache file")
}

// GetDefaultExchangeRates implements the CurrencyConverter interface.
// Gets exchange rates for the given symbols against the default base currency.
// Uses the file-backed cache if available and not expired; otherwise fetches from API.
func (cc *CachedCurrencyConverter) GetDefaultExchangeRates(ctx context.Context, symbols []string) ([]float32, error) {
	// Check cache first
	cc.mu.RLock()
	cacheRates, hit := cc.lookupCache()
	cc.mu.RUnlock()

	if hit && cacheRates != nil {
		cc.logger.Println("GetDefaultExchangeRates: cache hit, returning cached rates")
		// Return rates for requested symbols
		result := make([]float32, len(symbols))
		for i, sym := range symbols {
			if r, ok := cacheRates[sym]; ok {
				result[i] = r
			} else {
				result[i] = 0 // symbol not in cache
			}
		}
		return result, nil
	}

	cc.logger.Println("GetDefaultExchangeRates: cache miss, fetching from API")

	// Cache miss - fetch from API
	rates, err := cc.fetchFromAPI(symbols)
	if err != nil {
		cc.logger.Printf("GetDefaultExchangeRates: API fetch failed: %v", err)
		return nil, err
	}

	// Populate cache
	cc.mu.Lock()
	// Build rates map keyed by symbol
	ratesMap := make(map[string]float32)
	for i, sym := range symbols {
		ratesMap[sym] = rates[i]
	}
	cc.populateCache(ratesMap)
	cc.mu.Unlock()

	cc.logger.Println("GetDefaultExchangeRates: successfully populated cache")
	return rates, nil
}

// GetExchangeRates implements the CurrencyConverter interface.
// Gets exchange rates for the given symbols against the specified base currency.
// Uses the file-backed cache if available and not expired for the default base.
// For non-default bases, always fetches from API.
func (cc *CachedCurrencyConverter) GetExchangeRates(ctx context.Context, base string, symbols []string) ([]float32, error) {
	// If requesting rates against the default base, use cache
	if base == cc.defaultBase {
		cc.mu.RLock()
		cacheRates, hit := cc.lookupCache()
		cc.mu.RUnlock()

		if hit && cacheRates != nil {
			cc.logger.Println("GetExchangeRates: cache hit for default base")
			result := make([]float32, len(symbols))
			for i, sym := range symbols {
				if r, ok := cacheRates[sym]; ok {
					result[i] = r
				} else {
					result[i] = 0
				}
			}
			return result, nil
		}

		cc.logger.Println("GetExchangeRates: cache miss for default base, fetching from API")
		// Cache miss - fetch from API
		rates, err := cc.fetchFromAPI(symbols)
		if err != nil {
			cc.logger.Printf("GetExchangeRates: API fetch failed: %v", err)
			return nil, err
		}

		// Populate cache
		cc.mu.Lock()
		ratesMap := make(map[string]float32)
		for i, sym := range symbols {
			ratesMap[sym] = rates[i]
		}
		cc.populateCache(ratesMap)
		cc.mu.Unlock()

		cc.logger.Println("GetExchangeRates: successfully populated cache")
		return rates, nil
	}

	// Non-default base - always fetch from API
	cc.logger.Println("GetExchangeRates: non-default base, fetching from API")
	rates, err := cc.fetchFromAPI(symbols)
	if err != nil {
		cc.logger.Printf("GetExchangeRates: API fetch failed for non-default base: %v", err)
		return nil, err
	}

	return rates, nil
}

// fetchFromAPI fetches exchange rates from the API endpoint.
func (cc *CachedCurrencyConverter) fetchFromAPI(symbols []string) ([]float32, error) {
	// Build the URL with API key
	url := cc.apiBaseURL
	if url == "" {
		// Include the API key in the URL as required by the exchangerate API
		url = fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s?apikey=%s", cc.defaultBase, cc.defaultBase, cc.apiKey)
		cc.logger.Printf("fetchFromAPI: constructed API URL (masked): %s...%s", url[:40], url[len(url)-20:])
	}

	cc.logger.Printf("fetchFromAPI: making HTTP GET request to %s", url)

	// Make the HTTP request
	resp, err := http.Get(url)
	if err != nil {
		cc.logger.Printf("fetchFromAPI: HTTP request failed: %v", err)
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	cc.logger.Printf("fetchFromAPI: received response with status %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		bodyText := "unknown error"
		if resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			bodyText = string(b)
		}
		cc.logger.Printf("fetchFromAPI: API returned non-OK status %d, body: %s", resp.StatusCode, bodyText[:200])
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		cc.logger.Printf("fetchFromAPI: failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read api response: %w", err)
	}
	cc.logger.Printf("fetchFromAPI: read %d bytes from response body", len(body))

	// Parse the API response
	var apiResp struct {
		Result    string            `json:"result"`
		BaseCode  string            `json:"base_code"`
		Rates     map[string]float32 `json:"rates"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		cc.logger.Printf("fetchFromAPI: failed to parse API response JSON: %v", err)
		return nil, fmt.Errorf("failed to parse api response: %w", err)
	}

	cc.logger.Printf("fetchFromAPI: parsed API result: %s, base: %s, %d rate entries", apiResp.Result, apiResp.BaseCode, len(apiResp.Rates))

	if apiResp.Result != "success" {
		cc.logger.Printf("fetchFromAPI: API returned error result: %s", apiResp.Result)
		return nil, fmt.Errorf("api returned error result: %s", apiResp.Result)
	}

	// Build rates slice for requested symbols
	result := make([]float32, len(symbols))
	for i, sym := range symbols {
		if r, ok := apiResp.Rates[sym]; ok {
			result[i] = r
		} else {
			result[i] = 0 // symbol not available
		}
	}

	cc.logger.Printf("fetchFromAPI: built rates result with %d rates for symbols: %v", len(result), symbols)
	return result, nil
}