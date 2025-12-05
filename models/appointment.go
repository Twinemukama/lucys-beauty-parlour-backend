package models

type Appointment struct {
	ID            int64  `json:"id"`
	CustomerName  string `json:"customer_name" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	CustomerPhone string `json:"customer_phone" binding:"required"`
	StaffName     string `json:"staff_name"`
	Date          string `json:"date" binding:"required"`
	Time          string `json:"time" binding:"required"`
	Service       string `json:"service" binding:"required"`
	Notes         string `json:"notes,omitempty"`
	Status        string `json:"status" binding:"required"`
}
