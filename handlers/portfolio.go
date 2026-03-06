package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"lucys-beauty-parlour-backend/models"
	"lucys-beauty-parlour-backend/utils"

	"github.com/gin-gonic/gin"
)

// Allowed portfolio categories: hair | makeup | nails
func normalizeCategory(s string) string {
	switch s {
	case "hair", "makeup", "nails":
		return s
	default:
		return ""
	}
}

const maxImagesPerPortfolio = 10

func isPersistedImageRef(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	// Persisted refs are either data URIs or absolute URLs. Do not accept ephemeral /uploads paths.
	return strings.HasPrefix(v, "data:image/") || strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

func isLocalUploadPath(s string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "/uploads/")
}

// Public: list with filters (category, search query, pagination)
func (h *AppHandlers) ListPortfolioItems(c *gin.Context) {
	category := c.Query("category")
	if category != "" {
		category = normalizeCategory(category)
		if category == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category. Use one of: hair, makeup, nails"})
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

	items, total := h.Store.ListPortfolioItems(category, q, offset, limit)
	c.JSON(http.StatusOK, gin.H{
		"data":     items,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"has_more": offset+limit < total,
	})
}

// Public: get single
func (h *AppHandlers) GetPortfolioItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	it, err := h.Store.GetPortfolioItem(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, it)
}

// Admin: create
func (h *AppHandlers) CreatePortfolioItem(c *gin.Context) {
	var req models.PortfolioItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.Category = normalizeCategory(req.Category)
	if req.Category == "" {
		c.JSON(400, gin.H{"error": "invalid category. Use one of: hair, makeup, nails"})
		return
	}
	// Validate images and persist as durable data URIs (or keep existing refs)
	if len(req.Images) == 0 {
		c.JSON(400, gin.H{"error": "images are required"})
		return
	}
	if len(req.Images) > maxImagesPerPortfolio {
		c.JSON(400, gin.H{"error": "too many images", "max": maxImagesPerPortfolio})
		return
	}
	stored := make([]string, 0, len(req.Images))
	for i, img := range req.Images {
		if isValidBase64Image(img) {
			dataURI, err := utils.Base64ImageToDataURI(img)
			if err != nil {
				c.JSON(500, gin.H{"error": "failed to store image", "index": i})
				return
			}
			stored = append(stored, dataURI)
			continue
		}
		if isPersistedImageRef(img) {
			stored = append(stored, img)
			continue
		}
		c.JSON(400, gin.H{"error": "invalid image at index", "index": i})
		return
	}
	req.Images = stored
	created := h.Store.CreatePortfolioItem(&req)
	if created == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create portfolio item"})
		return
	}
	c.JSON(201, created)
}

// Admin: update
func (h *AppHandlers) UpdatePortfolioItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	var req models.PortfolioItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Category != "" {
		req.Category = normalizeCategory(req.Category)
		if req.Category == "" {
			c.JSON(400, gin.H{"error": "invalid category. Use one of: hair, makeup, nails"})
			return
		}
	}
	// Validate & persist images if provided, otherwise retain existing
	if req.Images != nil {
		if len(req.Images) > maxImagesPerPortfolio {
			c.JSON(400, gin.H{"error": "too many images", "max": maxImagesPerPortfolio})
			return
		}
		// fetch current to compute cleanup diff
		curr, _ := h.Store.GetPortfolioItem(id)
		stored := make([]string, 0, len(req.Images))
		for i, img := range req.Images {
			if isValidBase64Image(img) {
				dataURI, err := utils.Base64ImageToDataURI(img)
				if err != nil {
					c.JSON(500, gin.H{"error": "failed to store image", "index": i})
					return
				}
				stored = append(stored, dataURI)
				continue
			}
			if isPersistedImageRef(img) {
				stored = append(stored, img)
				continue
			}
			c.JSON(400, gin.H{"error": "invalid image at index", "index": i})
			return
		}
		req.Images = stored
		// cleanup old files not present anymore
		if curr != nil {
			old := make(map[string]bool, len(curr.Images))
			for _, p := range curr.Images {
				if isLocalUploadPath(p) {
					old[p] = true
				}
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
		curr, err := h.Store.GetPortfolioItem(id)
		if err == nil && curr != nil {
			req.Images = curr.Images
		}
	}
	upd, err := h.Store.UpdatePortfolioItem(id, &req)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, upd)
}

// Admin: delete
func (h *AppHandlers) DeletePortfolioItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	// fetch to cleanup images
	curr, _ := h.Store.GetPortfolioItem(id)
	if err := h.Store.DeletePortfolioItem(id); err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	if curr != nil {
		for _, p := range curr.Images {
			if isLocalUploadPath(p) {
				_ = utils.DeleteImageAndThumbnail(p)
			}
		}
	}
	c.Status(204)
}
