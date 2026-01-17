package types

type PriceResponse struct {
	Ticker string  `json:"ticker"`
	Price  float64 `json:"price"`
}

type BatchPriceResponse struct {
	Prices map[string]float64 `json:"prices"`
	Errors []string           `json:"errors,omitempty"`
}

type HistoricalPricePoint struct {
	Date  string  `json:"date"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

type HistoricalPriceResponse struct {
	Ticker string                 `json:"ticker"`
	Data   []HistoricalPricePoint `json:"data"`
}

type CreateAlertRequest struct {
	Ticker     string `json:"ticker"`
	Condition  string `json:"condition"`
	Threshold  float64 `json:"threshold"`
	WebhookURL string `json:"webhook_url"`
}

type ListAlertsResponse struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	ID          string     `json:"id"`
	Ticker      string     `json:"ticker"`
	Condition   string     `json:"condition"`
	Threshold   float64    `json:"threshold"`
	WebhookURL  string     `json:"webhook_url"`
	Active      bool       `json:"active"`
	CreatedAt   string     `json:"created_at"`
	TriggeredAt *string    `json:"triggered_at,omitempty"`
}
