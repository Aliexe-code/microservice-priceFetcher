package service

import (
	"context"
	"fmt"
	"testing"
)

func TestMockPriceFetcher(t *testing.T) {
	tests := []struct {
		name    string
		ticker  string
		want    float64
		wantErr bool
	}{
		{
			name:    "Valid AAPL ticker",
			ticker:  "AAPL",
			want:    150.0,
			wantErr: false,
		},
		{
			name:    "Valid MSFT ticker",
			ticker:  "MSFT",
			want:    300.0,
			wantErr: false,
		},
		{
			name:    "Valid GOOGL ticker",
			ticker:  "GOOGL",
			want:    2800.0,
			wantErr: false,
		},
		{
			name:    "Invalid ticker",
			ticker:  "INVALID",
			want:    0,
			wantErr: true,
		},
		{
			name:    "Empty ticker",
			ticker:  "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MockPriceFetcher(context.Background(), tt.ticker)
			if (err != nil) != tt.wantErr {
				t.Errorf("MockPriceFetcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MockPriceFetcher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceService_FetchPrice(t *testing.T) {
	svc := &priceService{}

	tests := []struct {
		name    string
		ticker  string
		want    float64
		wantErr bool
	}{
		{
			name:    "Fetch AAPL price",
			ticker:  "AAPL",
			want:    150.0,
			wantErr: false,
		},
		{
			name:    "Fetch MSFT price",
			ticker:  "MSFT",
			want:    300.0,
			wantErr: false,
		},
		{
			name:    "Fetch GOOGL price",
			ticker:  "GOOGL",
			want:    2800.0,
			wantErr: false,
		},
		{
			name:    "Fetch invalid ticker",
			ticker:  "INVALID",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.FetchPrice(context.Background(), tt.ticker)
			if (err != nil) != tt.wantErr {
				t.Errorf("priceService.FetchPrice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("priceService.FetchPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPriceService(t *testing.T) {
	tests := []struct {
		name       string
		useRealData string
		wantType   string
	}{
		{
			name:       "Mock data service",
			useRealData: "false",
			wantType:   "*main.priceService",
		},
		{
			name:       "Real data service",
			useRealData: "true",
			wantType:   "*main.AlphaVantageService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			t.Setenv("USE_REAL_DATA", tt.useRealData)

			svc := NewPriceService()

			// Check type
			typeName := fmt.Sprintf("%T", svc)
			// Update expected type names to match new package structure
			expectedType := tt.wantType
			if tt.wantType == "*main.priceService" {
				expectedType = "*service.priceService"
			} else if tt.wantType == "*main.AlphaVantageService" {
				expectedType = "*service.AlphaVantageService"
			}
			if typeName != expectedType {
				t.Errorf("NewPriceService() type = %v, want %v", typeName, expectedType)
			}
		})
	}
}