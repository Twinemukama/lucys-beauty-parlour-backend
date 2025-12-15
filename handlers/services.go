package handlers

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"lucys-beauty-parlour-backend/models"
	"lucys-beauty-parlour-backend/utils"

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

const maxImagesPerService = 8

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
	// Validate images are base64 and persist to uploads
	if len(req.Images) == 0 {
		c.JSON(400, gin.H{"error": "images are required and must be base64 strings"})
		return
	}
	if len(req.Images) > maxImagesPerService {
		c.JSON(400, gin.H{"error": "too many images", "max": maxImagesPerService})
		return
	}
	stored := make([]string, 0, len(req.Images))
	for i, img := range req.Images {
		if !isValidBase64Image(img) {
			c.JSON(400, gin.H{"error": "invalid base64 image at index", "index": i})
			return
		}
		path, err := utils.SaveBase64Image(img)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to store image", "index": i})
			return
		}
		stored = append(stored, path)
	}
	req.Images = stored
	created := h.Store.CreateServiceItem(&req)
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
	// Validate & persist images if provided, otherwise retain existing
	if req.Images != nil {
		if len(req.Images) > maxImagesPerService {
			c.JSON(400, gin.H{"error": "too many images", "max": maxImagesPerService})
			return
		}
		// fetch current to compute cleanup diff
		curr, _ := h.Store.GetServiceItem(id)
		stored := make([]string, 0, len(req.Images))
		for i, img := range req.Images {
			if !isValidBase64Image(img) {
				c.JSON(400, gin.H{"error": "invalid base64 image at index", "index": i})
				return
			}
			path, err := utils.SaveBase64Image(img)
			if err != nil {
				c.JSON(500, gin.H{"error": "failed to store image", "index": i})
				return
			}
			stored = append(stored, path)
		}
		req.Images = stored
		// cleanup old files not present anymore
		if curr != nil {
			old := make(map[string]bool, len(curr.Images))
			for _, p := range curr.Images {
				old[p] = true
			}
			for _, p := range stored {
				delete(old, p)
			}
			for p := range old {
				_ = utils.DeleteImageAndThumbnail(p)
			}
		}
	} else {
		// retain existing images when not provided
		curr, err := h.Store.GetServiceItem(id)
		if err == nil && curr != nil {
			req.Images = curr.Images
		}
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
	// fetch to cleanup images
	curr, _ := h.Store.GetServiceItem(id)
	if err := h.Store.DeleteServiceItem(id); err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	if curr != nil {
		for _, p := range curr.Images {
			_ = utils.DeleteImageAndThumbnail(p)
		}
	}
	c.Status(204)
}
