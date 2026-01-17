package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aliexe/ms-priceFetcher/internal/service"
	"github.com/aliexe/ms-priceFetcher/pkg/types"
	"github.com/google/uuid"
)

type APIFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error
type JSONAPIServer struct {
	svc        service.PriceService
	alertSvc   *service.AlertService
	listenAddr string
	server     *http.Server
}

func NewJSONAPIServer(listenAddr string, svc service.PriceService, alertSvc *service.AlertService) *JSONAPIServer {
	return &JSONAPIServer{
		svc:        svc,
		alertSvc:   alertSvc,
		listenAddr: listenAddr,
	}
}

func (s *JSONAPIServer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/price", makeHTTPHandler(s.handleFetchPrice))
	mux.HandleFunc("/prices", makeHTTPHandler(s.handleFetchPrices))
	mux.HandleFunc("/price/history", makeHTTPHandler(s.handleFetchPriceHistory))
	mux.HandleFunc("/alerts", s.handleAlerts)
	mux.HandleFunc("/alerts/", s.handleAlertByID)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
	}

	fmt.Println("Server started on", s.listenAddr)
	return s.server.ListenAndServe()
}

func (s *JSONAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *JSONAPIServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func makeHTTPHandler(apiFn APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := uuid.New().ID()
		ctx = context.WithValue(ctx, "requestID", reqID)
		if err := apiFn(ctx, w, r); err != nil {
			statusCode := http.StatusInternalServerError
			if isClientError(err) {
				statusCode = http.StatusBadRequest
			}
			http.Error(w, err.Error(), statusCode)
		}
	}
}

func isClientError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	clientErrorMessages := []string{
		"ticker is required",
		"invalid ticker",
		"ticker not found",
	}
	for _, msg := range clientErrorMessages {
		if contains(errMsg, msg) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *JSONAPIServer) handleFetchPrice(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		return fmt.Errorf("ticker is required")
	}

	// Validate ticker format
	if !isValidTicker(ticker) {
		return fmt.Errorf("invalid ticker format: must be 1-10 alphanumeric characters")
	}

	price, err := s.svc.FetchPrice(ctx, ticker)
	if err != nil {
		return err
	}
	priceResponse := types.PriceResponse{
		Ticker: ticker,
		Price:  price,
	}
	return writeJSON(w, http.StatusOK, priceResponse)
}

func (s *JSONAPIServer) handleFetchPrices(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	tickersParam := r.URL.Query().Get("tickers")
	if tickersParam == "" {
		return fmt.Errorf("tickers parameter is required")
	}

	// Parse comma-separated tickers
	tickers := parseTickers(tickersParam)
	if len(tickers) == 0 {
		return fmt.Errorf("at least one valid ticker is required")
	}

	// Validate all tickers
	for _, ticker := range tickers {
		if !isValidTicker(ticker) {
			return fmt.Errorf("invalid ticker format: %s must be 1-10 alphanumeric characters", ticker)
		}
	}

	// Limit to 50 tickers per request
	if len(tickers) > 50 {
		return fmt.Errorf("maximum 50 tickers per request")
	}

	prices, err := s.svc.FetchPrices(ctx, tickers)
	if err != nil {
		return err
	}

	batchResponse := types.BatchPriceResponse{
		Prices: prices,
	}
	return writeJSON(w, http.StatusOK, batchResponse)
}

func parseTickers(tickersParam string) []string {
	var tickers []string
	for _, ticker := range strings.Split(tickersParam, ",") {
		ticker = strings.TrimSpace(ticker)
		if ticker != "" {
			tickers = append(tickers, ticker)
		}
	}
	return tickers
}

func (s *JSONAPIServer) handleFetchPriceHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		return fmt.Errorf("ticker is required")
	}

	// Validate ticker format
	if !isValidTicker(ticker) {
		return fmt.Errorf("invalid ticker format: must be 1-10 alphanumeric characters")
	}

	fromDate := r.URL.Query().Get("from")
	toDate := r.URL.Query().Get("to")

	// Validate date format if provided
	if fromDate != "" && !isValidDate(fromDate) {
		return fmt.Errorf("invalid from date format: use YYYY-MM-DD")
	}
	if toDate != "" && !isValidDate(toDate) {
		return fmt.Errorf("invalid to date format: use YYYY-MM-DD")
	}

	history, err := s.svc.FetchPriceHistory(ctx, ticker, fromDate, toDate)
	if err != nil {
		return err
	}

	response := types.HistoricalPriceResponse{
		Ticker: ticker,
		Data:   history,
	}
	return writeJSON(w, http.StatusOK, response)
}

func isValidDate(date string) bool {
	if len(date) != 10 {
		return false
	}
	if date[4] != '-' || date[7] != '-' {
		return false
	}
	return true
}

func (s *JSONAPIServer) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleListAlerts(w, r)
	case "POST":
		s.handleCreateAlert(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *JSONAPIServer) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	var req types.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Ticker == "" {
		http.Error(w, "ticker is required", http.StatusBadRequest)
		return
	}

	if !isValidTicker(req.Ticker) {
		http.Error(w, "invalid ticker format", http.StatusBadRequest)
		return
	}

	if req.Condition != string(service.ConditionAbove) && req.Condition != string(service.ConditionBelow) {
		http.Error(w, "condition must be 'above' or 'below'", http.StatusBadRequest)
		return
	}

	if req.Threshold <= 0 {
		http.Error(w, "threshold must be positive", http.StatusBadRequest)
		return
	}

	alert, err := s.alertSvc.CreateAlert(req.Ticker, service.AlertCondition(req.Condition), req.Threshold, req.WebhookURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, alert)
}

func (s *JSONAPIServer) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := s.alertSvc.ListAlerts()

	// Convert to response format
	alertResponses := make([]types.Alert, len(alerts))
	for i, alert := range alerts {
		alertResponses[i] = types.Alert{
			ID:         alert.ID,
			Ticker:     alert.Ticker,
			Condition:  string(alert.Condition),
			Threshold:  alert.Threshold,
			WebhookURL: alert.WebhookURL,
			Active:     alert.Active,
			CreatedAt:  alert.CreatedAt.Format(time.RFC3339),
		}
		if alert.TriggeredAt != nil {
			triggeredAt := alert.TriggeredAt.Format(time.RFC3339)
			alertResponses[i].TriggeredAt = &triggeredAt
		}
	}

	response := types.ListAlertsResponse{
		Alerts: alertResponses,
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *JSONAPIServer) handleAlertByID(w http.ResponseWriter, r *http.Request) {
	alertID := strings.TrimPrefix(r.URL.Path, "/alerts/")

	switch r.Method {
	case "GET":
		alert, err := s.alertSvc.GetAlert(alertID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		alertResponse := types.Alert{
			ID:         alert.ID,
			Ticker:     alert.Ticker,
			Condition:  string(alert.Condition),
			Threshold:  alert.Threshold,
			WebhookURL: alert.WebhookURL,
			Active:     alert.Active,
			CreatedAt:  alert.CreatedAt.Format(time.RFC3339),
		}
		if alert.TriggeredAt != nil {
			triggeredAt := alert.TriggeredAt.Format(time.RFC3339)
			alertResponse.TriggeredAt = &triggeredAt
		}

		writeJSON(w, http.StatusOK, alertResponse)

	case "DELETE":
		if err := s.alertSvc.DeleteAlert(alertID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func isValidTicker(ticker string) bool {
	if len(ticker) < 1 || len(ticker) > 10 {
		return false
	}

	for _, r := range ticker {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return false
		}
	}

	return true
}

func writeJSON(w http.ResponseWriter, s int, v any) error {
	w.WriteHeader(s)
	return json.NewEncoder(w).Encode(v)
}
