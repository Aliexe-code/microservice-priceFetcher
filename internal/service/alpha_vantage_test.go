package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAlphaVantageService(t *testing.T) {
	svc := NewAlphaVantageService()

	if svc == nil {
		t.Fatal("NewAlphaVantageService() returned nil")
	}

	if svc.apiKey == "" {
		t.Error("Expected API key to be set")
	}

	if svc.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if svc.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestAlphaVantageService_FetchPrice(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		if r.URL.Query().Get("function") != "GLOBAL_QUOTE" {
			t.Error("Expected function parameter to be GLOBAL_QUOTE")
		}
		if r.URL.Query().Get("symbol") != "AAPL" {
			t.Error("Expected symbol parameter to be AAPL")
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "AAPL",
				"02. open": "149.5000",
				"03. high": "150.5000",
				"04. low": "149.0000",
				"05. price": "150.0000",
				"06. volume": "50000000",
				"07. latest trading day": "2024-01-01",
				"08. previous close": "148.0000",
				"09. change": "2.0000",
				"10. change percent": "1.35%"
			}
		}`))
	}))
	defer server.Close()

	// Create service with test server URL
	svc := &AlphaVantageService{
		apiKey:  "test-key",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    make(map[string]cacheEntry),
		cacheTTL: 5 * time.Minute,
	}

	// Test fetching price
	price, err := svc.FetchPrice(context.Background(), "AAPL")
	if err != nil {
		t.Errorf("FetchPrice() error = %v", err)
		return
	}

	if price != 150.0 {
		t.Errorf("FetchPrice() = %v, want 150.0", price)
	}
}

func TestAlphaVantageService_Cache(t *testing.T) {
	svc := NewAlphaVantageService()

	// Set cache entry
	svc.setCachedPrice("AAPL", 150.0)

	// Test cache hit
	price, found := svc.getCachedPrice("AAPL")
	if !found {
		t.Error("Expected cache hit for AAPL")
	}
	if price != 150.0 {
		t.Errorf("getCachedPrice() = %v, want 150.0", price)
	}

	// Test cache miss
	_, found = svc.getCachedPrice("INVALID")
	if found {
		t.Error("Expected cache miss for INVALID")
	}

	// Test cache expiration
	svc.cacheTTL = 1 * time.Millisecond
	svc.setCachedPrice("MSFT", 300.0)
	time.Sleep(10 * time.Millisecond)
	_, found = svc.getCachedPrice("MSFT")
	if found {
		t.Error("Expected cache to expire for MSFT")
	}
}

func TestAlphaVantageService_ClearCache(t *testing.T) {
	svc := NewAlphaVantageService()

	// Add cache entries
	svc.setCachedPrice("AAPL", 150.0)
	svc.setCachedPrice("MSFT", 300.0)

	// Clear cache
	svc.ClearCache()

	// Verify cache is cleared
	_, found := svc.getCachedPrice("AAPL")
	if found {
		t.Error("Expected cache to be cleared for AAPL")
	}

	_, found = svc.getCachedPrice("MSFT")
	if found {
		t.Error("Expected cache to be cleared for MSFT")
	}
}

func TestAlphaVantageService_APIError(t *testing.T) {
	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	svc := &AlphaVantageService{
		apiKey:  "test-key",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    make(map[string]cacheEntry),
		cacheTTL: 5 * time.Minute,
	}

	_, err := svc.FetchPrice(context.Background(), "AAPL")
	if err == nil {
		t.Error("Expected error for API failure")
	}
}

func TestAlphaVantageService_InvalidResponse(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ invalid json }`))
	}))
	defer server.Close()

	svc := &AlphaVantageService{
		apiKey:  "test-key",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    make(map[string]cacheEntry),
		cacheTTL: 5 * time.Minute,
	}

	_, err := svc.FetchPrice(context.Background(), "AAPL")
	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}

func TestAlphaVantageService_EnvironmentVariable(t *testing.T) {
	// Test with custom API key
	t.Setenv("ALPHA_VANTAGE_API_KEY", "custom-key")
	svc := NewAlphaVantageService()

	if svc.apiKey != "custom-key" {
		t.Errorf("Expected API key to be 'custom-key', got '%s'", svc.apiKey)
	}

	// Test with no API key (should default to "demo")
	t.Setenv("ALPHA_VANTAGE_API_KEY", "")
	svc = NewAlphaVantageService()

	if svc.apiKey != "demo" {
		t.Errorf("Expected API key to default to 'demo', got '%s'", svc.apiKey)
	}
}