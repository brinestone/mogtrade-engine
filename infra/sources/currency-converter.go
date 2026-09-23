package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExchangeRateSource is an interface for fetching exchange rates from a particular source.
// Implementations should prioritize bulk conversion methods when possible.
type ExchangeRateSource interface {
	// PullBulk fetches exchange rates for the given base currency and target currencies.
	// Returns a map of target currency -> rate.
	// This is the preferred method for fetching rates as it's more efficient.
	PullBulk(base string, currencies []string) (map[string]float32, error)

	// Pull fetches a single exchange rate from base to dest currency.
	// This is a fallback method; sources should implement PullBulk when possible.
	Pull(base, dest string) (float32, error)

	// Name returns the name of the source for logging/identification.
	Name() string
}

// CurrencyConverterConfig holds the configuration for NewCachedCurrencyConverter.
type CurrencyConverterConfig struct {
	// TTL in seconds for cache entries
	TTLSeconds int64

	// Default base currency (e.g., "USD")
	DefaultBase string

	// Logger for structured logging
	Logger *slog.Logger
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
	logger      *slog.Logger
	sources     []ExchangeRateSource
	sourceIndex int // for round-robin
}

// NewCachedCurrencyConverter creates a new CachedCurrencyConverter with the
// given params. The logger is required; the cache path uses an environment
// variable (TEMP or TMPDIR) for platform-agnostic temporary file storage.
// If multiple ExchangeRateSource implementations are provided, they will be
// used in a round-robin fashion.
func NewCachedCurrencyConverter(p CurrencyConverterConfig, s ...ExchangeRateSource) *CachedCurrencyConverter {
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

	cc := &CachedCurrencyConverter{
		ttlSeconds:  p.TTLSeconds,
		defaultBase: p.DefaultBase,
		logger:      p.Logger.With("service", "currency-converter"),
		sources:     s,
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

// getRatesFromSources attempts to fetch rates from the available sources.
// Uses round-robin strategy if multiple sources are configured.
// Prioritizes bulk conversion (PullBulk) and falls back to single currency (Pull) if needed.
// Returns the rates map (key: "BASE-DEST" format).
func (cc *CachedCurrencyConverter) getRatesFromSources(base string, currencies []string) (map[string]float32, error) {
	// First, check if we have a cached result that's still valid
	cacheRates, hit := cc.lookupCache()
	if hit && cacheRates != nil {
		cc.logger.Debug("getRatesFromSources: cache hit", "base", base, "currencies", currencies)
		return cacheRates, nil
	}

	if len(cc.sources) == 0 {
		return nil, fmt.Errorf("no exchange rate sources configured")
	}

	// Round-robin through sources
	idx := cc.sourceIndex
	for attempt := 0; attempt < len(cc.sources); attempt++ {
		sourceIndex := (idx + attempt) % len(cc.sources)
		source := cc.sources[sourceIndex]

		cc.logger.Debug("getRatesFromSources: trying source", "source", source.Name(), "base", base, "currencies", currencies)

		// Try bulk conversion first (preferred)
		bulkRates, err := source.PullBulk(base, currencies)
		if err == nil && len(bulkRates) > 0 {
			// Build result map with BASE-DEST keys
			rates := make(map[string]float32)
			for currency, rate := range bulkRates {
				rates[fmt.Sprintf("%s-%s", base, currency)] = rate
			}
			cc.sourceIndex = (sourceIndex + 1) % len(cc.sources)
			cc.logger.Debug("getRatesFromSources: bulk conversion succeeded", "source", source.Name(), "converted", len(rates))
			return rates, nil
		}

		if err != nil {
			cc.logger.Warn("getRatesFromSources: bulk conversion failed, falling back to single pulls", "source", source.Name(), "err", err)
		}

		// Fallback: try individual Pull for each currency
		rates := make(map[string]float32)
		successCount := 0
		for _, currency := range currencies {
			rate, err := source.Pull(base, currency)
			if err != nil {
				cc.logger.Error("could not pull exchange rate", "base", base, "dest", currency, "source", source.Name(), "err", err.Error())
				continue
			}
			rates[fmt.Sprintf("%s-%s", base, currency)] = rate
			successCount++
		}

		if successCount > 0 {
			cc.sourceIndex = (sourceIndex + 1) % len(cc.sources)
			cc.logger.Debug("getRatesFromSources: single pulls succeeded", "source", source.Name(), "converted", successCount)
			return rates, nil
		}

		cc.logger.Warn("getRatesFromSources: source returned no rates, trying next", "source", source.Name())
	}

	return nil, fmt.Errorf("all exchange rate sources failed for base %s", base)
}

// GetDefaultExchangeRates implements the CurrencyConverter interface.
// Gets exchange rates for the given symbols against the default base currency.
// Uses the file-backed cache if available and not expired; otherwise fetches
// from the configured sources.
func (cc *CachedCurrencyConverter) GetDefaultExchangeRates(ctx context.Context, symbols []string) ([]float32, error) {
	base := cc.defaultBase

	rates, err := cc.getRatesFromSources(base, symbols)
	if err != nil {
		cc.logger.Error("GetDefaultExchangeRates: failed to get rates from sources", "err", err)
		return nil, err
	}

	cc.logger.Debug("GetDefaultExchangeRates: returning rates from source", "symbols_count", len(symbols))
	result := make([]float32, len(symbols))
	for i, sym := range symbols {
		if r, ok := rates[fmt.Sprintf("%s-%s", base, sym)]; ok {
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
	if base == cc.defaultBase {
		rates, err := cc.getRatesFromSources(base, symbols)
		if err != nil {
			cc.logger.Error("GetExchangeRates: failed to get rates from source", "err", err)
			return nil, err
		}

		result := make([]float32, len(symbols))
		for i, sym := range symbols {
			if r, ok := rates[fmt.Sprintf("%s-%s", base, sym)]; ok {
				result[i] = r
			} else {
				result[i] = 0
			}
		}
		return result, nil
	}

	// Non-default base - always fetch from sources
	rates, err := cc.getRatesFromSources(base, symbols)
	if err != nil {
		cc.logger.Error("GetExchangeRates: failed to get rates from source for non-default base", "err", err)
		return nil, err
	}

	result := make([]float32, len(symbols))
	for i, sym := range symbols {
		if r, ok := rates[fmt.Sprintf("%s-%s", base, sym)]; ok {
			result[i] = r
		} else {
			result[i] = 0
		}
	}
	return result, nil
}