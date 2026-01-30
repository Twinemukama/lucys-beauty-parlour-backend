package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lucys-beauty-parlour-backend/models"
	"lucys-beauty-parlour-backend/storage"
	"lucys-beauty-parlour-backend/utils"

	"github.com/gin-gonic/gin"
)

type createAppointmentRequest struct {
	CustomerName         string  `json:"customer_name" binding:"required"`
	CustomerEmail        string  `json:"customer_email" binding:"required,email"`
	CustomerPhone        string  `json:"customer_phone" binding:"required"`
	StaffName            string  `json:"staff_name"`
	Date                 string  `json:"date"`
	AppointmentDate      string  `json:"appointment_date"`
	AppointmentDateAlt   string  `json:"appointmentDate"`
	Time                 string  `json:"time"`
	AppointmentTime      string  `json:"appointment_time"`
	AppointmentTimeAlt   string  `json:"appointmentTime"`
	ServiceID            int64   `json:"service_id" binding:"required"`
	ServiceDescription   string  `json:"service_description" binding:"required"`
	SelectedOptionIDs    []int64 `json:"selected_option_ids"`
	SelectedOptionIDsAlt []int64 `json:"selectedOptionIds"`
	Currency             string  `json:"currency,omitempty"`
	PriceCents           int64   `json:"price_cents" binding:"required"`
	Notes                string  `json:"notes,omitempty"`
	Status               string  `json:"status"`
}

type updateAppointmentRequest struct {
	CustomerName         *string  `json:"customer_name"`
	CustomerEmail        *string  `json:"customer_email"`
	CustomerPhone        *string  `json:"customer_phone"`
	StaffName            *string  `json:"staff_name"`
	Date                 *string  `json:"date"`
	Time                 *string  `json:"time"`
	ServiceID            *int64   `json:"service_id"`
	ServiceDescription   *string  `json:"service_description"`
	SelectedOptionIDs    *[]int64 `json:"selected_option_ids"`
	SelectedOptionIDsAlt *[]int64 `json:"selectedOptionIds"`
	Currency             *string  `json:"currency"`
	PriceCents           *int64   `json:"price_cents"`
	Notes                *string  `json:"notes"`
	Status               *string  `json:"status"`
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizeAppointmentDate(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("date is required")
	}

	// If the input starts with YYYY-MM-DD (including RFC3339 timestamps), keep the date portion.
	if len(raw) >= 10 && raw[4] == '-' && raw[7] == '-' {
		return raw[:10], nil
	}

	// Accept common date formats from frontends.
	layouts := []string{
		"02/01/2006", // dd/mm/yyyy
		"02-01-2006", // dd-mm-yyyy
		"2006/01/02", // yyyy/mm/dd
		"01/02/2006", // mm/dd/yyyy (fallback)
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("invalid date format; expected YYYY-MM-DD")
}

func normalizeAppointmentTime(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("time is required")
	}

	layouts := []string{
		"15:04",
		"15:04:05",
		"3:04 PM",
		"03:04 PM",
		"3:04PM",
		"03:04PM",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t.Format("15:04"), nil
		}
	}

	return "", fmt.Errorf("invalid time format; expected HH:MM")
}

type AppHandlers struct {
	Store *storage.InMemoryStore
}

func (h *AppHandlers) CreateAppointment(c *gin.Context) {
	var req createAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dateRaw := firstNonEmpty(req.Date, req.AppointmentDate, req.AppointmentDateAlt)
	timeRaw := firstNonEmpty(req.Time, req.AppointmentTime, req.AppointmentTimeAlt)
	date, err := normalizeAppointmentDate(dateRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	clock, err := normalizeAppointmentTime(timeRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appointment := models.Appointment{
		CustomerName:       req.CustomerName,
		CustomerEmail:      req.CustomerEmail,
		CustomerPhone:      req.CustomerPhone,
		StaffName:          req.StaffName,
		Date:               date,
		Time:               clock,
		ServiceID:          req.ServiceID,
		ServiceDescription: req.ServiceDescription,
		Currency:           strings.TrimSpace(req.Currency),
		PriceCents:         req.PriceCents,
		Notes:              req.Notes,
		Status:             strings.TrimSpace(req.Status),
	}

	// Validate service exists by ServiceID (foreign key)
	if appointment.ServiceID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_id is required and must be positive"})
		return
	}
	svc, err := h.Store.GetServiceItem(appointment.ServiceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id: service not found"})
		return
	}

	// Validate selected description/style is provided and belongs to the service
	if appointment.ServiceDescription == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_description is required"})
		return
	}
	validDesc := false
	for _, d := range svc.Descriptions {
		if d == appointment.ServiceDescription {
			validDesc = true
			break
		}
	}
	if !validDesc {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_description for the selected service"})
		return
	}

	// Frontend provides the total price.
	if appointment.PriceCents < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price_cents must be >= 0"})
		return
	}

	// Check if the date has availability (max 15 confirmed appointments per day)
	if !h.Store.IsAppointmentSlotAvailable(appointment.Date) {
		c.JSON(http.StatusConflict, gin.H{"error": "No slots available for the requested date. Maximum appointments reached for the day."})
		return
	}

	// Set default status to "pending" if not provided
	if appointment.Status == "" {
		appointment.Status = "pending"
	}

	created := h.Store.CreateAppointment(&appointment)

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
	// Merge semantics: fetch current appointment first.
	curr, err := h.Store.GetAppointment(id)
	if err != nil || curr == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req updateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merged := *curr
	if req.CustomerName != nil {
		merged.CustomerName = strings.TrimSpace(*req.CustomerName)
	}
	if req.CustomerEmail != nil {
		merged.CustomerEmail = strings.TrimSpace(*req.CustomerEmail)
	}
	if req.CustomerPhone != nil {
		merged.CustomerPhone = strings.TrimSpace(*req.CustomerPhone)
	}
	if req.StaffName != nil {
		merged.StaffName = strings.TrimSpace(*req.StaffName)
	}
	if req.Date != nil {
		normalizedDate, err := normalizeAppointmentDate(strings.TrimSpace(*req.Date))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		merged.Date = normalizedDate
	}
	if req.Time != nil {
		normalizedTime, err := normalizeAppointmentTime(strings.TrimSpace(*req.Time))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		merged.Time = normalizedTime
	}
	if req.ServiceID != nil {
		merged.ServiceID = *req.ServiceID
	}
	if req.ServiceDescription != nil {
		merged.ServiceDescription = strings.TrimSpace(*req.ServiceDescription)
	}
	// Ignore selected_option_ids for now (frontend-calculated total price is what we persist).
	if req.Notes != nil {
		merged.Notes = *req.Notes
	}
	if req.Status != nil {
		merged.Status = strings.TrimSpace(*req.Status)
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

	// If service or description are present, validate description belongs to service
	svcForValidation := (*models.ServiceItem)(nil)
	if merged.ServiceID > 0 {
		if svc, err := h.Store.GetServiceItem(merged.ServiceID); err == nil {
			svcForValidation = svc
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id: service not found"})
			return
		}
	}
	if merged.ServiceDescription != "" {
		// Determine service to validate against
		if svcForValidation == nil {
			if svc, err := h.Store.GetServiceItem(merged.ServiceID); err == nil {
				svcForValidation = svc
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current service for validation"})
				return
			}
		}
		validDesc := false
		for _, d := range svcForValidation.Descriptions {
			if d == merged.ServiceDescription {
				validDesc = true
				break
			}
		}
		if !validDesc {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_description for the selected service"})
			return
		}
	}

	updated, err := h.Store.UpdateAppointment(id, &merged)
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
		if err := utils.SendAppointmentRejectedEmail(appt, serviceName); err != nil {
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
