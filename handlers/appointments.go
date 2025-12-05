package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"lucys-beauty-parlour-backend/models"
	"lucys-beauty-parlour-backend/storage"
	"lucys-beauty-parlour-backend/utils"

	"github.com/gin-gonic/gin"
)

type AppHandlers struct {
	Store *storage.InMemoryStore
}

func (h *AppHandlers) CreateAppointment(c *gin.Context) {
	var req models.Appointment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the date has availability (max 15 confirmed appointments per day)
	if !h.Store.IsAppointmentSlotAvailable(req.Date) {
		c.JSON(http.StatusConflict, gin.H{"error": "No slots available for the requested date. Maximum appointments reached for the day."})
		return
	}

	// Set default status to "pending" if not provided
	if req.Status == "" {
		req.Status = "pending"
	}

	created := h.Store.CreateAppointment(&req)

	// Send notification email to admin asynchronously
	go func() {
		if err := utils.SendNewAppointmentNotificationToAdmin(created); err != nil {
			fmt.Println("Error sending admin notification:", err)
		}
	}()

	c.JSON(http.StatusCreated, created)
}

func (h *AppHandlers) ListAppointments(c *gin.Context) {
	// Get pagination parameters from query
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	appointments, totalCount := h.Store.GetAppointmentsWithPagination(offset, limit)

	c.JSON(http.StatusOK, gin.H{
		"data":     appointments,
		"total":    totalCount,
		"offset":   offset,
		"limit":    limit,
		"has_more": offset+limit < totalCount,
	})
}

func (h *AppHandlers) GetAppointment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	a, err := h.Store.GetAppointment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AppHandlers) UpdateAppointment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.Appointment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.Store.UpdateAppointment(id, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Send confirmation or update email to customer asynchronously
	go func() {
		if updated.Status == "confirmed" {
			if err := utils.SendAppointmentConfirmedEmail(updated); err != nil {
				fmt.Println("Error sending confirmation email:", err)
			}
		} else {
			if err := utils.SendAppointmentUpdatedEmail(updated); err != nil {
				fmt.Println("Error sending update email:", err)
			}
		}
	}()

	c.JSON(http.StatusOK, updated)
}

func (h *AppHandlers) CancelAppointment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Get appointment before cancelling to send email
	appointment, err := h.Store.GetAppointment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	updated, err := h.Store.CancelAppointment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Send rejection/cancellation email to customer asynchronously
	go func() {
		if err := utils.SendAppointmentRejectedEmail(appointment.CustomerEmail, appointment.CustomerName, id); err != nil {
			fmt.Println("Error sending rejection email:", err)
		}
	}()

	c.JSON(http.StatusOK, updated)
}

func (h *AppHandlers) DeleteAppointment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.Store.DeleteAppointment(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
