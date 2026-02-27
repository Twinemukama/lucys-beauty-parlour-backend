package models

type ServiceItem struct {
	ID           int64    `json:"id"`
	Service      string   `json:"service" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Descriptions []string `json:"descriptions" binding:"required"`
	Rating       float64  `json:"rating"`
}
