package handlers

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"lucys-beauty-parlour-backend/models"

	"github.com/gin-gonic/gin"
)

// Allowed services: hair | makeup | nails
func normalizeService(s string) string {
	switch s {
	case "hair", "makeup", "nails":
		return s
	default:
		return ""
	}
}

// isValidBase64Image checks whether the provided string is valid base64 data.
// Supports optional data URI prefix like: data:image/png;base64,XXXXX
func isValidBase64Image(s string) bool {
	if s == "" {
		return false
	}
	// Strip data URI prefix if present
	if idx := strings.Index(s, ","); idx != -1 && strings.Contains(strings.ToLower(s[:idx]), "base64") {
		s = s[idx+1:]
	}
	// Base64 decode validation
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		// Try RawStdEncoding as some inputs may omit padding
		if _, err2 := base64.RawStdEncoding.DecodeString(s); err2 != nil {
			return false
		}
	}
	return true
}

// Public: list with filters (category, min_rating, pagination)
func (h *AppHandlers) ListServiceItems(c *gin.Context) {
	category := c.Query("category")
	if category != "" {
		category = normalizeService(category)
		if category == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category. Use one of: hair, makeup, nails"})
			return
		}
	}

	minRating := 0.0
	if v := c.Query("min_rating"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minRating = f
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid min_rating"})
			return
		}
	}

	q := c.Query("q")

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

	items, total := h.Store.ListServiceItems(category, minRating, q, offset, limit)
	c.JSON(http.StatusOK, gin.H{
		"data":     items,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"has_more": offset+limit < total,
	})
}

// Public: get single
func (h *AppHandlers) GetServiceItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	it, err := h.Store.GetServiceItem(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, it)
}

// Admin: create
func (h *AppHandlers) CreateServiceItem(c *gin.Context) {
	var req models.ServiceItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.Service = normalizeService(req.Service)
	if req.Service == "" {
		c.JSON(400, gin.H{"error": "invalid service. Use one of: hair, makeup, nails"})
		return
	}
	if req.Rating < 0 || req.Rating > 5 {
		c.JSON(400, gin.H{"error": "rating must be between 0 and 5"})
		return
	}
	created := h.Store.CreateServiceItem(&req)
	if created == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create service item"})
		return
	}
	c.JSON(201, created)
}

// Admin: update
func (h *AppHandlers) UpdateServiceItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	var req models.ServiceItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Service != "" {
		req.Service = normalizeService(req.Service)
		if req.Service == "" {
			c.JSON(400, gin.H{"error": "invalid service. Use one of: hair, makeup, nails"})
			return
		}
	}
	if req.Rating < 0 || req.Rating > 5 {
		c.JSON(400, gin.H{"error": "rating must be between 0 and 5"})
		return
	}
	upd, err := h.Store.UpdateServiceItem(id, &req)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, upd)
}

// Admin: delete
func (h *AppHandlers) DeleteServiceItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if err := h.Store.DeleteServiceItem(id); err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.Status(204)
}
