package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AlertCondition represents the type of alert condition
type AlertCondition string

const (
	ConditionAbove AlertCondition = "above"
	ConditionBelow AlertCondition = "below"
)

// Alert represents a price alert configuration
type Alert struct {
	ID          string         `json:"id"`
	Ticker      string         `json:"ticker"`
	Condition   AlertCondition `json:"condition"`
	Threshold   float64        `json:"threshold"`
	WebhookURL  string         `json:"webhook_url"`
	Active      bool           `json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	TriggeredAt *time.Time     `json:"triggered_at,omitempty"`
}

// AlertService manages price alerts
type AlertService struct {
	alerts      map[string]*Alert
	alertsMutex sync.RWMutex
	priceSvc    PriceService
	httpClient  *http.Client
	logger      *logrus.Logger
}

// NewAlertService creates a new alert service
func NewAlertService(priceSvc PriceService) *AlertService {
	return &AlertService{
		alerts:   make(map[string]*Alert),
		priceSvc: priceSvc,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logrus.New(),
	}
}

// CreateAlert creates a new price alert
func (s *AlertService) CreateAlert(ticker string, condition AlertCondition, threshold float64, webhookURL string) (*Alert, error) {
	s.alertsMutex.Lock()
	defer s.alertsMutex.Unlock()

	alertID := fmt.Sprintf("%s-%s-%.2f", ticker, condition, threshold)
	alert := &Alert{
		ID:         alertID,
		Ticker:     ticker,
		Condition:  condition,
		Threshold:  threshold,
		WebhookURL: webhookURL,
		Active:     true,
		CreatedAt:  time.Now(),
	}

	s.alerts[alertID] = alert
	s.logger.WithFields(logrus.Fields{
		"alertID":   alertID,
		"ticker":    ticker,
		"condition": condition,
		"threshold": threshold,
	}).Info("Alert created")

	return alert, nil
}

// GetAlert retrieves an alert by ID
func (s *AlertService) GetAlert(alertID string) (*Alert, error) {
	s.alertsMutex.RLock()
	defer s.alertsMutex.RUnlock()

	alert, exists := s.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	return alert, nil
}

// ListAlerts returns all alerts
func (s *AlertService) ListAlerts() []*Alert {
	s.alertsMutex.RLock()
	defer s.alertsMutex.RUnlock()

	alerts := make([]*Alert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// DeleteAlert removes an alert
func (s *AlertService) DeleteAlert(alertID string) error {
	s.alertsMutex.Lock()
	defer s.alertsMutex.Unlock()

	if _, exists := s.alerts[alertID]; !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	delete(s.alerts, alertID)
	s.logger.WithField("alertID", alertID).Info("Alert deleted")

	return nil
}

// CheckAlerts evaluates all active alerts against current prices
func (s *AlertService) CheckAlerts(ctx context.Context) error {
	s.alertsMutex.RLock()
	defer s.alertsMutex.RUnlock()

	for _, alert := range s.alerts {
		if !alert.Active {
			continue
		}

		price, err := s.priceSvc.FetchPrice(ctx, alert.Ticker)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"alertID": alert.ID,
				"ticker":  alert.Ticker,
				"error":   err,
			}).Error("Failed to fetch price for alert check")
			continue
		}

		triggered := false
		switch alert.Condition {
		case ConditionAbove:
			triggered = price > alert.Threshold
		case ConditionBelow:
			triggered = price < alert.Threshold
		}

		if triggered {
			s.logger.WithFields(logrus.Fields{
				"alertID":   alert.ID,
				"ticker":    alert.Ticker,
				"price":     price,
				"threshold": alert.Threshold,
				"condition": alert.Condition,
			}).Info("Alert triggered")

			// Send webhook notification
			if err := s.sendWebhook(alert, price); err != nil {
				s.logger.WithFields(logrus.Fields{
					"alertID": alert.ID,
					"error":   err,
				}).Error("Failed to send webhook")
			}

			// Mark alert as triggered
			now := time.Now()
			alert.TriggeredAt = &now
			alert.Active = false
		}
	}

	return nil
}

// StartAlertChecker starts a background goroutine to check alerts periodically
func (s *AlertService) StartAlertChecker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.WithField("interval", interval).Info("Starting alert checker")

	for {
		select {
		case <-ticker.C:
			if err := s.CheckAlerts(ctx); err != nil {
				s.logger.WithError(err).Error("Alert check failed")
			}
		case <-ctx.Done():
			s.logger.Info("Stopping alert checker")
			return
		}
	}
}

// sendWebhook sends a webhook notification for a triggered alert
func (s *AlertService) sendWebhook(alert *Alert, price float64) error {
	if alert.WebhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"alert_id":   alert.ID,
		"ticker":     alert.Ticker,
		"condition":  alert.Condition,
		"threshold":  alert.Threshold,
		"current_price": price,
		"triggered_at": time.Now().Format(time.RFC3339),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", alert.WebhookURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = nil // Reset body

	req, err = http.NewRequest("POST", alert.WebhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}