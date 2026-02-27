package models

type PortfolioItem struct {
	ID          int64    `json:"id"`
	Category    string   `json:"category" binding:"required"` // hair, makeup, nails
	Style       string   `json:"style" binding:"required"`    // Box braids, Soft Glam, etc.
	Images      []string `json:"images" binding:"required"`
	Description string   `json:"description" binding:"required"`
	CreatedAt   string   `json:"created_at"`
}
