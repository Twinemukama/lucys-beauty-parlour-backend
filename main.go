package main

import (
	"os"

	"lucys-beauty-parlour-backend/handlers"
	"lucys-beauty-parlour-backend/middleware"
	"lucys-beauty-parlour-backend/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// Allow Gin mode to be controlled via GIN_MODE env, default to release.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	store := storage.NewInMemoryStore()
	h := &handlers.AppHandlers{Store: store}

	// Public routes
	r.POST("/admin/login", handlers.AdminLogin)
	r.POST("/admin/forgot-password", handlers.ForgotPassword)
	r.POST("/admin/change-password", handlers.ChangePassword)
	r.POST("/appointments", h.CreateAppointment)

	// Protected routes (admin only)
	admin := r.Group("/", middleware.AdminAuth())
	{
		admin.GET("/appointments", h.ListAppointments)
		admin.GET("/appointments/:id", h.GetAppointment)
		admin.PUT("/appointments/:id", h.UpdateAppointment)
		admin.PUT("/appointments/:id/cancel", h.CancelAppointment)
		admin.DELETE("/appointments/:id", h.DeleteAppointment)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	r.Run(":" + port)
}
