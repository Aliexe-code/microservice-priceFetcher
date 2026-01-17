package service

import (
	"context"
	"fmt"
	"os"

	"github.com/aliexe/ms-priceFetcher/pkg/types"
)

var priceMocks = map[string]float64{
	"AAPL":  150.0,
	"MSFT":  300.0,
	"GOOGL": 2800.0,
}

type priceService struct{}

type PriceService interface {
	FetchPrice(context.Context, string) (float64, error)
	FetchPrices(context.Context, []string) (map[string]float64, error)
	FetchPriceHistory(context.Context, string, string, string) ([]types.HistoricalPricePoint, error)
}

func (s *priceService) FetchPrice(ctx context.Context, ticker string) (float64, error) {
	return MockPriceFetcher(ctx, ticker)
}

func (s *priceService) FetchPrices(ctx context.Context, tickers []string) (map[string]float64, error) {
	return MockBatchPriceFetcher(ctx, tickers)
}

func (s *priceService) FetchPriceHistory(ctx context.Context, ticker, fromDate, toDate string) ([]types.HistoricalPricePoint, error) {
	return MockPriceHistoryFetcher(ctx, ticker, fromDate, toDate)
}

func MockPriceFetcher(ctx context.Context, ticker string) (float64, error) {
	price, ok := priceMocks[ticker]
	if !ok {
		return 0, fmt.Errorf("price not found for %s", ticker)
	}
	return price, nil
}

func MockBatchPriceFetcher(ctx context.Context, tickers []string) (map[string]float64, error) {
	results := make(map[string]float64)
	var errors []error

	for _, ticker := range tickers {
		price, err := MockPriceFetcher(ctx, ticker)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		results[ticker] = price
	}

	if len(results) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to fetch prices for any ticker: %v", errors)
	}

	return results, nil
}

func MockPriceHistoryFetcher(ctx context.Context, ticker, fromDate, toDate string) ([]types.HistoricalPricePoint, error) {
	if _, ok := priceMocks[ticker]; !ok {
		return nil, fmt.Errorf("ticker not found: %s", ticker)
	}

	// Generate mock historical data
	basePrice := priceMocks[ticker]
	mockData := []types.HistoricalPricePoint{
		{Date: "2024-01-01", Open: basePrice - 5, High: basePrice + 5, Low: basePrice - 10, Close: basePrice - 2},
		{Date: "2024-01-02", Open: basePrice - 2, High: basePrice + 3, Low: basePrice - 5, Close: basePrice + 1},
		{Date: "2024-01-03", Open: basePrice + 1, High: basePrice + 8, Low: basePrice - 3, Close: basePrice + 5},
		{Date: "2024-01-04", Open: basePrice + 5, High: basePrice + 10, Low: basePrice, Close: basePrice + 3},
		{Date: "2024-01-05", Open: basePrice + 3, High: basePrice + 7, Low: basePrice + 1, Close: basePrice + 6},
	}

	return mockData, nil
}

// NewPriceService creates a price service based on environment configuration
// Set USE_REAL_DATA=true to use Alpha Vantage API, otherwise uses mock data
func NewPriceService() PriceService {
	useRealData := os.Getenv("USE_REAL_DATA") == "true"

	if useRealData {
		return NewAlphaVantageService()
	}

	return &priceService{}
}
