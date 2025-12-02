package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "lucys-beauty-parlour-backend/models"
    "lucys-beauty-parlour-backend/storage"
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
    created := h.Store.CreateAppointment(&req)
    c.JSON(http.StatusCreated, created)
}

func (h *AppHandlers) ListAppointments(c *gin.Context) {
    all := h.Store.GetAllAppointments()
    c.JSON(http.StatusOK, all)
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
