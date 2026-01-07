package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"lucys-beauty-parlour-backend/models"

	"github.com/gin-gonic/gin"
)

func normalizeMenuCategory(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return s
}

type createMenuItemRequest struct {
	Category        string `json:"category" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Currency        string `json:"currency"`
	PriceCents      int64  `json:"price_cents" binding:"required"`
	DurationMinutes int    `json:"duration_minutes" binding:"required"`
}

type updateMenuItemRequest struct {
	Category        *string `json:"category"`
	Name            *string `json:"name"`
	Currency        *string `json:"currency"`
	PriceCents      *int64  `json:"price_cents"`
	DurationMinutes *int    `json:"duration_minutes"`
}

// Public: list menu items
func (h *AppHandlers) ListMenuItems(c *gin.Context) {
	category := c.Query("category")
	category = normalizeMenuCategory(category)

	q := strings.TrimSpace(c.Query("q"))

	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
	}
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
	}

	items, total := h.Store.ListMenuItems(category, q, offset, limit)
	c.JSON(http.StatusOK, gin.H{
		"data":     items,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"has_more": offset+limit < total,
	})
}

// Public: get one
func (h *AppHandlers) GetMenuItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	it, err := h.Store.GetMenuItem(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, it)
}

// Admin: create
func (h *AppHandlers) CreateMenuItem(c *gin.Context) {
	var req createMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := normalizeMenuCategory(req.Category)
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category is required"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if req.PriceCents < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price_cents must be >= 0"})
		return
	}

	if req.DurationMinutes <= 0 || req.DurationMinutes > 24*60 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "duration_minutes must be between 1 and 1440"})
		return
	}

	item := &models.MenuItem{
		Category:        category,
		Name:            name,
		Currency:        strings.TrimSpace(req.Currency),
		PriceCents:      req.PriceCents,
		DurationMinutes: req.DurationMinutes,
	}

	created := h.Store.CreateMenuItem(item)
	c.JSON(http.StatusCreated, created)
}

// Admin: update (partial)
func (h *AppHandlers) UpdateMenuItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	curr, err := h.Store.GetMenuItem(id)
	if err != nil || curr == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req updateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merged := *curr
	if req.Category != nil {
		merged.Category = normalizeMenuCategory(*req.Category)
		if merged.Category == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category must be non-empty"})
			return
		}
	}
	if req.Name != nil {
		merged.Name = strings.TrimSpace(*req.Name)
		if merged.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
	}
	if req.Currency != nil {
		merged.Currency = strings.TrimSpace(*req.Currency)
	}
	if req.PriceCents != nil {
		if *req.PriceCents < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "price_cents must be >= 0"})
			return
		}
		merged.PriceCents = *req.PriceCents
	}
	if req.DurationMinutes != nil {
		if *req.DurationMinutes <= 0 || *req.DurationMinutes > 24*60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duration_minutes must be between 1 and 1440"})
			return
		}
		merged.DurationMinutes = *req.DurationMinutes
	}

	upd, err := h.Store.UpdateMenuItem(id, &merged)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, upd)
}

// Admin: delete
func (h *AppHandlers) DeleteMenuItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.Store.DeleteMenuItem(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
