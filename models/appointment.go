package models

type Appointment struct {
	ID                 int64  `json:"id"`
	CustomerName       string `json:"customer_name" binding:"required"`
	CustomerEmail      string `json:"customer_email" binding:"required,email"`
	CustomerPhone      string `json:"customer_phone" binding:"required"`
	StaffName          string `json:"staff_name"`
	Date               string `json:"date" binding:"required"`
	Time               string `json:"time" binding:"required"`
	ServiceID          int64  `json:"service_id" binding:"required"`
	ServiceDescription string `json:"service_description" binding:"required"`

	Currency   string `json:"currency,omitempty"`
	PriceCents int64  `json:"price_cents"`

	Notes  string `json:"notes,omitempty"`
	Status string `json:"status"`
}
