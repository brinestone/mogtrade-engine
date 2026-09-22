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

// currencySource is an interface for fetching exchange rates from a particular source.
type currencySource interface {
	// Pull fetches exchange rates for the given base and currencies.
	// Returns a map of currency -> rate.
	Pull(base string, currencies []string) (map[string]float32, error)
}

// params holds the configuration for NewCachedCurrencyConverter.
type params struct {
	// TTL in seconds for cache entries
	TTLSeconds int64

	// Default base currency (e.g., "USD")
	DefaultBase string

	// API base URL (without API key)
	APIBaseURL string

	// API key for the exchange rate service
	APIKey string

	// Logger for structured logging
	Logger *slog.Logger

	// Sources is a slice of CurrencySource implementations.
	// If multiple sources are provided, the converter will use a
	// round-robin strategy to fetch from them, which helps save
	// on token/rate limits by distributing requests.
	Sources []currencySource
}

// CachedCurrencyConverter implements the CurrencyConverter interface using
// a file-backed cache for exchange rates. On cache-miss, the converter
// fetches rates from the configured sources and stores them in the
// cache with a configurable TTL.
type CachedCurrencyConverter struct {
	mu          sync.RWMutex
	cacheFile   *os.File
	cachePath   string // path to cache file
	ttlSeconds  int64
	defaultBase string
	apiBaseURL  string
	apiKey      string
	logger      *slog.Logger
	sources     []currencySource // nil means use single-source fallback
	sourceIndex int              // for round-robin
}

// NewCachedCurrencyConverter creates a new CachedCurrencyConverter with the
// given params. The logger is required; the cache path uses an environment
// variable (TEMP or TMPDIR) for platform-agnostic temporary file storage.
// If multiple CurrencySource implementations are provided, they will be
// used in a round-robin fashion.
func NewCachedCurrencyConverter(p params) *CachedCurrencyConverter {
	// Validate required fields
	if p.Logger == nil {
		panic("currency-converter: logger is required")
	}

	// Set defaults
	if p.TTLSeconds <= 0 {
		p.TTLSeconds = 300 // 5 minutes default
	}
	if p.DefaultBase == "" {
		p.DefaultBase = "USD"
	}
	if p.APIBaseURL == "" {
		p.APIBaseURL = "https://v6.exchangerate-api.com/v6"
	}

	cc := &CachedCurrencyConverter{
		ttlSeconds:  p.TTLSeconds,
		defaultBase: p.DefaultBase,
		apiBaseURL:  p.APIBaseURL,
		apiKey:      p.APIKey,
		logger:      p.Logger,
		sources:     p.Sources,
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

// getRateFromSources attempts to fetch rates from the available sources.
// Uses round-robin strategy if multiple sources are configured.
// Returns the rates map and the source index used (for round-robin).
func (cc *CachedCurrencyConverter) getRateFromSources(base string, currencies []string) (map[string]float32, int, error) {
	// First, check if we have a cached result that's still valid
	cacheRates, hit := cc.lookupCache()
	if hit && cacheRates != nil {
		return cacheRates, cc.sourceIndex, nil
	}

	var rates map[string]float32
	var err error

	// If no sources configured, or single source, use direct fetch
	if len(cc.sources) == 0 {
		// Fall back to direct API call using the embedded URL pattern
		rates, err = cc.fetchFromAPI(base, currencies)
	} else {
		// Round-robin through sources
		// Start from current index, advance after use
	 idx := cc.sourceIndex
	 // Try each source until one succeeds or we've tried them all
	 for attempt := 0; attempt < len(cc.sources); attempt++ {
		 sourceIdx := (idx + attempt) % len(cc.sources)
		 rates, err = cc.sources[sourceIdx].Pull(base, currencies)
		 if err == nil && rates != nil {
			 // Success! Update source index for next time
			 cc.sourceIndex = (sourceIdx + 1) % len(cc.sources)
			 return rates, sourceIdx, nil
		 }
	 }
	 // If all sources failed, return error
	 return nil, cc.sourceIndex, err
	}

	// If we got here via the no-sources path, update the source index
	// (keep it unchanged since we didn't use round-robin)
	if len(cc.sources) == 0 {
		// No sources, just return what we have
	}

	return rates, cc.sourceIndex, err
}

// GetDefaultExchangeRates implements the CurrencyConverter interface.
// Gets exchange rates for the given symbols against the default base currency.
// Uses the file-backed cache if available and not expired; otherwise fetches
// from the configured sources.
func (cc *CachedCurrencyConverter) GetDefaultExchangeRates(ctx context.Context, symbols []string) ([]float32, error) {
	// Determine the base currency from the symbols (first symbol is typically the base)
	// or use the configured default base.
	base := cc.defaultBase

	// Try to get rates from sources (with cache check and round-robin)
	rates, _, err := cc.getRateFromSources(base, symbols)
	if err != nil {
		cc.logger.Error("GetDefaultExchangeRates: failed to get rates from sources", "err", err)
		return nil, err
	}

	// Build result slice aligned with the requested symbols
	cc.logger.Debug("GetDefaultExchangeRates: returning rates from source", "symbols_count", len(symbols))
	result := make([]float32, len(symbols))
	for i, sym := range symbols {
		if r, ok := rates[sym]; ok {
			result[i] = r
		} else {
			result[i] = 0 // symbol not available
		}
	}
	return result, nil
}

// GetExchangeRates implements the CurrencyConverter interface.
// Gets exchange rates for the given symbols against the specified base currency.
// Uses the file-backed cache if available and not expired for the default base.
// For non-default bases, always fetches from the configured sources.
func (cc *CachedCurrencyConverter) GetExchangeRates(ctx context.Context, base string, symbols []string) ([]float32, error) {
	// If requesting rates against the default base, use sources
	if base == cc.defaultBase {
		rates, _, err := cc.getRateFromSources(base, symbols)
		if err != nil {
			cc.logger.Error("GetExchangeRates: failed to get rates from source", "err", err)
			return nil, err
		}

		result := make([]float32, len(symbols))
		for i, sym := range symbols {
			if r, ok := rates[sym]; ok {
				result[i] = r
			} else {
				result[i] = 0
			}
		}
		return result, nil
	}

	// Non-default base - always fetch from sources
	rates, _, err := cc.getRateFromSources(base, symbols)
	if err != nil {
		cc.logger.Error("GetExchangeRates: failed to get rates from source for non-default base", "err", err)
		return nil, err
	}

	result := make([]float32, len(symbols))
	for i, sym := range symbols {
		if r, ok := rates[sym]; ok {
			result[i] = r
		} else {
			result[i] = 0
		}
	}
	return result, nil
}

// fetchFromAPI fetches exchange rates from the API endpoint.
// Used as a fallback when no external sources are configured.
func (cc *CachedCurrencyConverter) fetchFromAPI(base string, currencies []string) (map[string]float32, error) {
	// Build the URL
	url := cc.apiBaseURL
	if url == "" {
		url = fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s?apikey=%s", cc.defaultBase, cc.defaultBase, cc.apiKey)
	}

	// Make the HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read api response: %w", err)
	}

	// Parse the API response
	var apiResp struct {
		Result    string            `json:"result"`
		BaseCode  string            `json:"base_code"`
		Rates     map[string]float32 `json:"rates"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse api response: %w", err)
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("api returned error result: %s", apiResp.Result)
	}

	// Filter to only the requested currencies
	result := make(map[string]float32)
	for _, sym := range currencies {
		if r, ok := apiResp.Rates[sym]; ok {
			result[sym] = r
		}
	}

	return result, nil
}