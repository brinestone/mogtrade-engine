package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"log/slog"
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
	logger      *slog.Logger
}

// NewCachedCurrencyConverter creates a new CachedCurrencyConverter.
// The logger is injected from outside; the cache path uses an environment
// variable (TEMP or TMPDIR) for platform-agnostic temporary file storage.
func NewCachedCurrencyConverter(ttlSeconds int64, defaultBase string, apiBaseURL string, apiKey string, logger *slog.Logger) *CachedCurrencyConverter {
	cc := &CachedCurrencyConverter{
		ttlSeconds:  ttlSeconds,
		defaultBase: defaultBase,
		apiBaseURL:  apiBaseURL,
		apiKey:      apiKey,
		logger:      logger,
	}
	cc.initializeCache()
	return cc
}

// initializeCache opens/creates the cache file using a platform-agnostic temp directory.
func (cc *CachedCurrencyConverter) initializeCache() {
	// Determine temp directory from environment variable, platform-agnostic
	tempDir := os.Getenv("TEMP")
	if tempDir == "" {
		// Try TMPDIR (macOS/Linux)
		tempDir = os.Getenv("TMPDIR")
	}
	if tempDir == "" {
		// Fallback to /tmp
		tempDir = "/tmp"
	}

	// Ensure the temp directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		cc.logger.Error("failed to create temp directory", "dir", tempDir, "err", err)
		tempDir = "/tmp"
	}

	// Construct cache file path
	cc.cachePath = filepath.Join(tempDir, "mogtrade-currency-cache.json")

	// Create/truncate the cache file
	var err error
	cc.cacheFile, err = os.Create(cc.cachePath)
	if err != nil {
		cc.logger.Error("failed to create cache file", "path", cc.cachePath, "err", err)
		// Fallback: try current directory
		cc.cachePath = "currency-cache.json"
		cc.cacheFile, err = os.Create(cc.cachePath)
		if err != nil {
			cc.logger.Error("failed to create fallback cache file", "path", cc.cachePath, "err", err)
			cc.cacheFile = nil
		}
	}
}

// closeCache closes the cache file.
func (cc *CachedCurrencyConverter) closeCache() {
	if cc.cacheFile != nil {
		cc.logger.Debug("closing cache file", "path", cc.cachePath)
		cc.cacheFile.Close()
	}
}

// lookupCache checks the file-backed cache for rates against the default base.
// Returns the rates map and whether the cache entry is valid (not expired).
func (cc *CachedCurrencyConverter) lookupCache() (map[string]float32, bool) {
	if cc.cacheFile == nil {
		cc.logger.Debug("lookupCache: cache file not available, returning cache miss")
		return nil, false
	}

	data, err := os.ReadFile(cc.cachePath)
	if err != nil {
		cc.logger.Debug("lookupCache: failed to read cache file", "path", cc.cachePath, "err", err)
		return nil, false
	}

	var rates map[string]float32
	if err := json.Unmarshal(data, &rates); err != nil {
		cc.logger.Debug("lookupCache: failed to unmarshal cache JSON", "err", err)
		return nil, false
	}

	// Check TTL: verify the cache was written recently
	now := time.Now().Unix()
	fileInfo, _ := os.Stat(cc.cachePath)
	fileModTime := fileInfo.ModTime().Unix()

	// If cache file was modified within TTL, consider it valid
	if now-fileModTime <= cc.ttlSeconds && len(rates) > 0 {
		cc.logger.Debug("lookupCache: cache hit", "ttl_seconds_remaining", cc.ttlSeconds-(now-fileModTime), "rates_count", len(rates))
		return rates, true
	}

	cc.logger.Debug("lookupCache: cache miss", "ttl_seconds_old", now-fileModTime, "rates_count", len(rates))
	return nil, false
}

// populateCache writes the given rates to the cache file.
func (cc *CachedCurrencyConverter) populateCache(rates map[string]float32) {
	data, err := json.Marshal(rates)
	if err != nil {
		cc.logger.Error("populateCache: failed to marshal rates to JSON", "err", err)
		return
	}

	// Write to cache file
	if err := os.WriteFile(cc.cachePath, data, 0644); err != nil {
		cc.logger.Error("populateCache: failed to write cache file", "path", cc.cachePath, "err", err)
		return
	}
	cc.logger.Debug("populateCache: successfully wrote rates to cache file", "path", cc.cachePath)
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
		cc.logger.Debug("GetDefaultExchangeRates: cache hit, returning cached rates", "symbols_count", len(symbols))
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

	cc.logger.Debug("GetDefaultExchangeRates: cache miss, fetching from API")

	// Cache miss - fetch from API
	rates, err := cc.fetchFromAPI(symbols)
	if err != nil {
		cc.logger.Error("GetDefaultExchangeRates: API fetch failed", "err", err)
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

	cc.logger.Debug("GetDefaultExchangeRates: successfully populated cache", "rates_count", len(rates))
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
			cc.logger.Debug("GetExchangeRates: cache hit for default base", "symbols_count", len(symbols))
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

		cc.logger.Debug("GetExchangeRates: cache miss for default base, fetching from API")
		// Cache miss - fetch from API
		rates, err := cc.fetchFromAPI(symbols)
		if err != nil {
			cc.logger.Error("GetExchangeRates: API fetch failed", "err", err)
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

		cc.logger.Debug("GetExchangeRates: successfully populated cache", "rates_count", len(rates))
		return rates, nil
	}

	// Non-default base - always fetch from API
	cc.logger.Debug("GetExchangeRates: non-default base, fetching from API")
	rates, err := cc.fetchFromAPI(symbols)
	if err != nil {
		cc.logger.Error("GetExchangeRates: API fetch failed for non-default base", "err", err)
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
		cc.logger.Debug("fetchFromAPI: constructed API URL (masked)", "base", cc.defaultBase)
	}

	cc.logger.Debug("fetchFromAPI: making HTTP GET request", "url_masked", url[:min(40, len(url))]+"...")

	// Make the HTTP request
	resp, err := http.Get(url)
	if err != nil {
		cc.logger.Error("fetchFromAPI: HTTP request failed", "err", err)
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	cc.logger.Debug("fetchFromAPI: received response with status", "status_code", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		bodyText := "unknown error"
		if resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			bodyText = string(b)
		}
		cc.logger.Error("fetchFromAPI: API returned non-OK status", "status_code", resp.StatusCode, "body_snippet", string(bodyText)[:200])
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		cc.logger.Error("fetchFromAPI: failed to read response body", "err", err)
		return nil, fmt.Errorf("failed to read api response: %w", err)
	}
	cc.logger.Debug("fetchFromAPI: read response body", "body_bytes", len(body))

	// Parse the API response
	var apiResp struct {
		Result    string            `json:"result"`
		BaseCode  string            `json:"base_code"`
		Rates     map[string]float32 `json:"rates"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		cc.logger.Error("fetchFromAPI: failed to parse API response JSON", "err", err)
		return nil, fmt.Errorf("failed to parse api response: %w", err)
	}

	cc.logger.Debug("fetchFromAPI: parsed API result", "result", apiResp.Result, "base_code", apiResp.BaseCode, "rates_count", len(apiResp.Rates))

	if apiResp.Result != "success" {
		cc.logger.Error("fetchFromAPI: API returned error result", "result", apiResp.Result)
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

	cc.logger.Debug("fetchFromAPI: built rates result", "rates_count", len(result), "symbols", symbols)
	return result, nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}