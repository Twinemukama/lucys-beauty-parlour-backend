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

	// Validate service exists by ServiceID (foreign key)
	if req.ServiceID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_id is required and must be positive"})
		return
	}
	svc, err := h.Store.GetServiceItem(req.ServiceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id: service not found"})
		return
	}

	// Validate selected description/style is provided and belongs to the service
	if req.ServiceDescription == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_description is required"})
		return
	}
	validDesc := false
	for _, d := range svc.Descriptions {
		if d == req.ServiceDescription {
			validDesc = true
			break
		}
	}
	if !validDesc {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_description for the selected service"})
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

	// Lookup service name for notifications
	svcName := svc.Name

	// Send notification email to admin asynchronously
	go func(app *models.Appointment, serviceName string) {
		if err := utils.SendNewAppointmentNotificationToAdmin(app, serviceName); err != nil {
			fmt.Println("Error sending admin notification:", err)
		}
	}(created, svcName)

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
	// If service or description are present, validate description belongs to service
	svcForValidation := (*models.ServiceItem)(nil)
	if req.ServiceID > 0 {
		if svc, err := h.Store.GetServiceItem(req.ServiceID); err == nil {
			svcForValidation = svc
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id: service not found"})
			return
		}
	}
	if req.ServiceDescription != "" {
		// Determine service to validate against
		if svcForValidation == nil {
			// fetch current appointment to know service
			curr, err := h.Store.GetAppointment(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			if svc, err := h.Store.GetServiceItem(curr.ServiceID); err == nil {
				svcForValidation = svc
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current service for validation"})
				return
			}
		}
		validDesc := false
		for _, d := range svcForValidation.Descriptions {
			if d == req.ServiceDescription {
				validDesc = true
				break
			}
		}
		if !validDesc {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_description for the selected service"})
			return
		}
	}

	updated, err := h.Store.UpdateAppointment(id, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Lookup service name for emails
	svcName := ""
	if svc, err := h.Store.GetServiceItem(updated.ServiceID); err == nil {
		svcName = svc.Name
	}

	// Send confirmation or update email to customer asynchronously
	go func(app *models.Appointment, serviceName string) {
		if app.Status == "confirmed" {
			if err := utils.SendAppointmentConfirmedEmail(app, serviceName); err != nil {
				fmt.Println("Error sending confirmation email:", err)
			}
		} else {
			if err := utils.SendAppointmentUpdatedEmail(app, serviceName); err != nil {
				fmt.Println("Error sending update email:", err)
			}
		}
	}(updated, svcName)

	c.JSON(http.StatusOK, updated)
}

func (h *AppHandlers) CancelAppointment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Fetch appointment before cancelling to get contact and service information
	appointment, err := h.Store.GetAppointment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	updated, err := h.Store.CancelAppointment(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel appointment"})
		return
	}

	// Send cancellation email including service name
	go func(appt *models.Appointment) {
		serviceName := ""
		if svc, err := h.Store.GetServiceItem(appt.ServiceID); err == nil && svc != nil {
			serviceName = svc.Name
		}
		if err := utils.SendAppointmentRejectedEmail(appt.CustomerEmail, appt.CustomerName, serviceName, appt.ID); err != nil {
			fmt.Println("Error sending rejection email:", err)
		}
	}(appointment)

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
