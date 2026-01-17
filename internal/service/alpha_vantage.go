package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/aliexe/ms-priceFetcher/pkg/types"
)

// AlphaVantageResponse represents the API response from Alpha Vantage
type AlphaVantageResponse struct {
	GlobalQuote GlobalQuote `json:"Global Quote"`
}

// GlobalQuote contains the stock price data
type GlobalQuote struct {
	Symbol           string `json:"01. symbol"`
	Open             string `json:"02. open"`
	High             string `json:"03. high"`
	Low              string `json:"04. low"`
	Price            string `json:"05. price"`
	Volume           string `json:"06. volume"`
	LatestTradingDay string `json:"07. latest trading day"`
	PreviousClose    string `json:"08. previous close"`
	Change           string `json:"09. change"`
	ChangePercent    string `json:"10. change percent"`
}

// AlphaVantageService implements real-time stock price fetching
type AlphaVantageService struct {
	apiKey        string
	baseURL       string
	httpClient    *http.Client
	cache         map[string]cacheEntry
	cacheMutex    sync.RWMutex
	cacheTTL      time.Duration
	maxCacheSize  int
}

type cacheEntry struct {
	price    float64
	history  []types.HistoricalPricePoint
	expiry   time.Time
}
const (
	defaultMaxCacheSize = 1000 // Maximum number of cached entries
)

// NewAlphaVantageService creates a new Alpha Vantage service instance
func NewAlphaVantageService() *AlphaVantageService {
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	if apiKey == "" {
		apiKey = "demo" // Demo key for testing
	}

	// Configure HTTP transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &AlphaVantageService{
		apiKey:  apiKey,
		baseURL: "https://www.alphavantage.co/query",
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		cache:         make(map[string]cacheEntry),
		cacheTTL:      5 * time.Minute, // Cache prices for 5 minutes
		maxCacheSize:  defaultMaxCacheSize,
	}
}

// FetchPrice retrieves the current stock price from Alpha Vantage API
func (s *AlphaVantageService) FetchPrice(ctx context.Context, ticker string) (float64, error) {
	// Check cache first
	if price, found := s.getCachedPrice(ticker); found {
		return price, nil
	}

	// Build request URL
	params := url.Values{}
	params.Set("function", "GLOBAL_QUOTE")
	params.Set("symbol", ticker)
	params.Set("apikey", s.apiKey)

	reqURL := fmt.Sprintf("%s?%s", s.baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var avResponse AlphaVantageResponse
	if err := json.Unmarshal(body, &avResponse); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract price
	if avResponse.GlobalQuote.Price == "" {
		return 0, fmt.Errorf("invalid response: price field is empty for ticker %s", ticker)
	}

	var price float64
	_, err = fmt.Sscanf(avResponse.GlobalQuote.Price, "%f", &price)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}

	// Cache the price
	s.setCachedPrice(ticker, price)

	return price, nil
}

// FetchPrices retrieves multiple stock prices concurrently
func (s *AlphaVantageService) FetchPrices(ctx context.Context, tickers []string) (map[string]float64, error) {
	results := make(map[string]float64)
	resultsMutex := sync.Mutex{}
	errors := make([]error, 0)
	errorsMutex := sync.Mutex{}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent requests

	for _, ticker := range tickers {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			price, err := s.FetchPrice(ctx, t)
			if err != nil {
				errorsMutex.Lock()
				errors = append(errors, err)
				errorsMutex.Unlock()
				return
			}

			resultsMutex.Lock()
			results[t] = price
			resultsMutex.Unlock()
		}(ticker)
	}

	wg.Wait()

	if len(results) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to fetch prices for any ticker: %v", errors)
	}

	return results, nil
}

// FetchPriceHistory retrieves historical price data for a ticker
func (s *AlphaVantageService) FetchPriceHistory(ctx context.Context, ticker, fromDate, toDate string) ([]types.HistoricalPricePoint, error) {
	// Check cache first with a unique key
	cacheKey := fmt.Sprintf("history_%s_%s_%s", ticker, fromDate, toDate)
	if cached, found := s.getCachedHistory(cacheKey); found {
		return cached, nil
	}

	// Build request URL for TIME_SERIES_DAILY
	params := url.Values{}
	params.Set("function", "TIME_SERIES_DAILY")
	params.Set("symbol", ticker)
	params.Set("apikey", s.apiKey)
	params.Set("outputsize", "full")

	reqURL := fmt.Sprintf("%s?%s", s.baseURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var avResponse map[string]interface{}
	if err := json.Unmarshal(body, &avResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract time series data
	timeSeries, ok := avResponse["Time Series (Daily)"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response: time series data not found for ticker %s", ticker)
	}

	// Parse historical data points
	var historicalData []types.HistoricalPricePoint
	for date, data := range timeSeries {
		dataPoint, ok := data.(map[string]interface{})
		if !ok {
			continue
		}

		open, _ := parsePrice(dataPoint["1. open"])
		high, _ := parsePrice(dataPoint["2. high"])
		low, _ := parsePrice(dataPoint["3. low"])
		close, _ := parsePrice(dataPoint["4. close"])

		// Filter by date range if specified
		if fromDate != "" && date < fromDate {
			continue
		}
		if toDate != "" && date > toDate {
			continue
		}

		historicalData = append(historicalData, types.HistoricalPricePoint{
			Date:  date,
			Open:  open,
			High:  high,
			Low:   low,
			Close: close,
		})
	}

	// Sort by date (descending order in API, reverse to ascending)
	for i, j := 0, len(historicalData)-1; i < j; i, j = i+1, j-1 {
		historicalData[i], historicalData[j] = historicalData[j], historicalData[i]
	}

	// Cache the results
	s.setCachedHistory(cacheKey, historicalData)

	return historicalData, nil
}

func parsePrice(value interface{}) (float64, error) {
	var price float64
	switch v := value.(type) {
	case string:
		_, err := fmt.Sscanf(v, "%f", &price)
		return price, err
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid price type")
	}
}

// getCachedHistory retrieves cached historical data
func (s *AlphaVantageService) getCachedHistory(key string) ([]types.HistoricalPricePoint, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, exists := s.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiry) {
		return nil, false
	}

	return entry.history, true
}

// setCachedHistory stores historical data in cache
func (s *AlphaVantageService) setCachedHistory(key string, history []types.HistoricalPricePoint) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Evict expired entries first
	s.evictExpired()

	// If still at capacity, evict oldest entries
	if len(s.cache) >= s.maxCacheSize {
		s.evictOldest()
	}

	s.cache[key] = cacheEntry{
		history: history,
		expiry:  time.Now().Add(s.cacheTTL),
	}
}

// getCachedPrice retrieves a price from cache if it's still valid
func (s *AlphaVantageService) getCachedPrice(ticker string) (float64, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, exists := s.cache[ticker]
	if !exists {
		return 0, false
	}

	if time.Now().After(entry.expiry) {
		return 0, false
	}

	return entry.price, true
}

// setCachedPrice stores a price in cache with TTL
func (s *AlphaVantageService) setCachedPrice(ticker string, price float64) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Evict expired entries first
	s.evictExpired()

	// If still at capacity, evict oldest entries
	if len(s.cache) >= s.maxCacheSize {
		s.evictOldest()
	}

	s.cache[ticker] = cacheEntry{
		price:  price,
		expiry: time.Now().Add(s.cacheTTL),
	}
}

// evictExpired removes all expired entries from the cache
func (s *AlphaVantageService) evictExpired() {
	now := time.Now()
	for ticker, entry := range s.cache {
		if now.After(entry.expiry) {
			delete(s.cache, ticker)
		}
	}
}

// evictOldest removes approximately 10% of oldest entries when cache is full
func (s *AlphaVantageService) evictOldest() {
	if len(s.cache) == 0 {
		return
	}

	// Collect all entries with their expiry times
	type tickerExpiry struct {
		ticker string
		expiry time.Time
	}

	entries := make([]tickerExpiry, 0, len(s.cache))
	for ticker, entry := range s.cache {
		entries = append(entries, tickerExpiry{ticker, entry.expiry})
	}

	// Sort by expiry (oldest first)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].expiry.After(entries[j].expiry) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove approximately 10% of entries
	numToRemove := len(entries) / 10
	if numToRemove < 1 {
		numToRemove = 1
	}

	for i := 0; i < numToRemove; i++ {
		delete(s.cache, entries[i].ticker)
	}
}

// ClearCache clears all cached prices
func (s *AlphaVantageService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache = make(map[string]cacheEntry)
}