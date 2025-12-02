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

	// Public admin login
	r.POST("/admin/login", handlers.AdminLogin)

	// Protected routes
	admin := r.Group("/", middleware.AdminAuth())
	{
		admin.POST("/appointments", h.CreateAppointment)
		admin.GET("/appointments", h.ListAppointments)
		admin.GET("/appointments/:id", h.GetAppointment)
		admin.PUT("/appointments/:id", h.UpdateAppointment)
		admin.DELETE("/appointments/:id", h.DeleteAppointment)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
