package main

import (
	"log"
	"os"
	"time"

	"lucys-beauty-parlour-backend/database"
	"lucys-beauty-parlour-backend/handlers"
	"lucys-beauty-parlour-backend/middleware"
	"lucys-beauty-parlour-backend/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env for local/dev runs.
	// In production, env vars are typically injected by the runtime.
	_ = godotenv.Load()

	// Allow Gin mode to be controlled via GIN_MODE env, default to release.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.OpenFromEnv()
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
	if err := database.Seed(db); err != nil {
		log.Fatalf("failed to seed database: %v", err)
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

	store := storage.NewPostgresStore(db)
	h := &handlers.AppHandlers{Store: store}

	refreshStore := storage.NewRefreshStore()
	handlers.RefreshDB = refreshStore
	handlers.AdminDB = db

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
	// Menu items (public)
	r.GET("/menu-items", h.ListMenuItems)
	r.GET("/menu-items/:id", h.GetMenuItem)

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

		// Menu items (admin CRUD)
		admin.POST("/menu-items", h.CreateMenuItem)
		admin.PUT("/menu-items/:id", h.UpdateMenuItem)
		admin.DELETE("/menu-items/:id", h.DeleteMenuItem)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	r.Run(":" + port)
}
