package models

type MenuItem struct {
	ID              int64  `json:"id"`
	Category        string `json:"category"`
	Name            string `json:"name"`
	Currency        string `json:"currency,omitempty"`
	PriceCents      int64  `json:"price_cents"`
	DurationMinutes int    `json:"duration_minutes"`
}
