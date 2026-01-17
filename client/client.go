package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aliexe/ms-priceFetcher/proto"
	"github.com/aliexe/ms-priceFetcher/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents the HTTP client for the price fetcher service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new HTTP client for the price fetcher service
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchPrice retrieves the price for a given ticker symbol via HTTP
func (c *Client) FetchPrice(ctx context.Context, ticker string) (*types.PriceResponse, error) {
	url := fmt.Sprintf("%s/price?ticker=%s", c.baseURL, ticker)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var priceResponse types.PriceResponse
	if err := json.Unmarshal(body, &priceResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &priceResponse, nil
}

// NewGRPCClient creates a new gRPC client for the price fetcher service
func NewGRPCClient(addr string) (proto.PriceFetcherClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return proto.NewPriceFetcherClient(conn), nil
}