package main

import (
	"os"
	"time"

	"lucys-beauty-parlour-backend/handlers"
	"lucys-beauty-parlour-backend/middleware"
	"lucys-beauty-parlour-backend/models"
	"lucys-beauty-parlour-backend/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func seedServiceItems(store *storage.InMemoryStore) {
	services := []*models.ServiceItem{
		{
			ID:           1,
			Service:      "Hair Styling & Braiding",
			Name:         "Knotless Braids",
			Descriptions: []string{"Small", "Medium", "Large"},
			Images:       []string{},
			Rating:       0,
		},
		{
			ID:           2,
			Service:      "Hair Styling & Braiding",
			Name:         "Wig Install",
			Descriptions: []string{"Closure", "Frontal"},
			Images:       []string{},
			Rating:       0,
		},
		{
			ID:           3,
			Service:      "Makeup",
			Name:         "Soft Glam",
			Descriptions: []string{"Day", "Evening"},
			Images:       []string{},
			Rating:       0,
		},
		{
			ID:           4,
			Service:      "Makeup",
			Name:         "Bridal Makeup",
			Descriptions: []string{"Bride", "Bridesmaid"},
			Images:       []string{},
			Rating:       0,
		},
		{
			ID:           5,
			Service:      "Nails",
			Name:         "Gel Manicure",
			Descriptions: []string{"Short", "Medium", "Long"},
			Images:       []string{},
			Rating:       0,
		},
		{
			ID:           6,
			Service:      "Nails",
			Name:         "Acrylic Full Set",
			Descriptions: []string{"Short", "Medium", "Long"},
			Images:       []string{},
			Rating:       0,
		},
	}

	for _, service := range services {
		store.CreateServiceItem(service)
	}
}

func main() {
	// Load environment variables from .env for local/dev runs.
	// In production, env vars are typically injected by the runtime.
	_ = godotenv.Load()

	// Allow Gin mode to be controlled via GIN_MODE env, default to release.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "https://lucysbeautyparlour.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Serve uploaded files statically
	r.Static("/uploads", "./uploads")

	store := storage.NewInMemoryStore()
	h := &handlers.AppHandlers{Store: store}

	refreshStore := storage.NewRefreshStore()
	handlers.RefreshDB = refreshStore

	// Seed service items
	seedServiceItems(store)

	// Public routes
	r.POST("/admin/login", handlers.AdminLogin)
	r.POST("/admin/refresh", handlers.RefreshToken)
	r.POST("/admin/logout", handlers.Logout)
	r.POST("/admin/forgot-password", handlers.ForgotPassword)
	r.POST("/admin/change-password", handlers.ChangePassword)
	r.POST("/appointments", h.CreateAppointment)
	// Services blog (public)
	r.GET("/services", h.ListServiceItems)
	r.GET("/services/:id", h.GetServiceItem)

	// Protected routes (admin only)
	admin := r.Group("/admin", middleware.AdminAuth())
	{
		admin.GET("/appointments", h.ListAppointments)
		admin.GET("/appointments/:id", h.GetAppointment)
		admin.PUT("/appointments/:id", h.UpdateAppointment)
		admin.PUT("/appointments/:id/cancel", h.CancelAppointment)
		admin.DELETE("/appointments/:id", h.DeleteAppointment)

		// Services blog (admin CRUD)
		admin.POST("/services", h.CreateServiceItem)
		admin.PUT("/services/:id", h.UpdateServiceItem)
		admin.DELETE("/services/:id", h.DeleteServiceItem)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	r.Run(":" + port)
}
