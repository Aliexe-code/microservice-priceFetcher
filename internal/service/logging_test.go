package service

import (
	"context"
	"testing"
	"time"

	"github.com/aliexe/ms-priceFetcher/pkg/types"
)

type mockPriceService struct {
	price  float64
	err    error
	prices map[string]float64
}

func (m *mockPriceService) FetchPrice(ctx context.Context, ticker string) (float64, error) {
	return m.price, m.err
}

func (m *mockPriceService) FetchPrices(ctx context.Context, tickers []string) (map[string]float64, error) {
	if m.prices != nil {
		return m.prices, m.err
	}
	return nil, m.err
}

func (m *mockPriceService) FetchPriceHistory(ctx context.Context, ticker, fromDate, toDate string) ([]types.HistoricalPricePoint, error) {
	return []types.HistoricalPricePoint{}, m.err
}

func TestLoggingService_FetchPrice(t *testing.T) {
	tests := []struct {
		name    string
		price   float64
		err     error
		wantErr bool
	}{
		{
			name:    "Successful fetch",
			price:   150.0,
			err:     nil,
			wantErr: false,
		},
		{
			name:    "Failed fetch",
			price:   0,
			err:     context.Canceled,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockPriceService{
				price: tt.price,
				err:   tt.err,
			}

			loggingSvc := NewLoggingService(mockSvc)

			// Add request ID to context
			ctx := context.WithValue(context.Background(), "requestID", 12345)

			got, err := loggingSvc.FetchPrice(ctx, "AAPL")

			if (err != nil) != tt.wantErr {
				t.Errorf("LoggingService.FetchPrice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.price {
				t.Errorf("LoggingService.FetchPrice() = %v, want %v", got, tt.price)
			}

			// Ensure logging doesn't block the call for too long
			// (This is a basic sanity check)
			start := time.Now()
			loggingSvc.FetchPrice(ctx, "AAPL")
			duration := time.Since(start)

			if duration > 100*time.Millisecond {
				t.Errorf("LoggingService.FetchPrice() took too long: %v", duration)
			}
		})
	}
}

func TestNewLoggingService(t *testing.T) {
	mockSvc := &mockPriceService{price: 150.0, err: nil}

	loggingSvc := NewLoggingService(mockSvc)

	if loggingSvc == nil {
		t.Fatal("NewLoggingService() returned nil")
	}

	if loggingSvc.(*LoggingService).next == nil {
		t.Error("Expected next service to be set")
	}
}