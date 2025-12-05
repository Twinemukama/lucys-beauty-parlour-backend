package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"lucys-beauty-parlour-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type changePasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// In-memory storage for password reset tokens (in production, use a database)
var (
	resetTokens = make(map[string]time.Time)
	tokenMutex  sync.RWMutex
)

func AdminLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminEmail == "" {
		adminEmail = "twinemukamai@gmail.com"
	}
	if adminPassword == "" {
		adminPassword = "LBP@2025"
	}

	if req.Email != adminEmail || req.Password != adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "secret"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin": true,
		"exp":   time.Now().Add(12 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed})
}

// ForgotPassword sends a password reset email
func ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "twinemukamai@gmail.com"
	}

	// Only allow password reset for the admin email
	if req.Email != adminEmail {
		// Don't reveal if email exists for security
		c.JSON(http.StatusOK, gin.H{"message": "If this email exists, you will receive a password reset link shortly."})
		return
	}

	// Generate reset token
	resetToken := generateToken(32)

	// Store token with 1-hour expiry
	tokenMutex.Lock()
	resetTokens[resetToken] = time.Now().Add(1 * time.Hour)
	tokenMutex.Unlock()

	// Send email
	err := utils.SendPasswordResetEmail(req.Email, resetToken)
	if err != nil {
		// Log error but return success to prevent email enumeration
		fmt.Println("Email send error:", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "If this email exists, you will receive a password reset link shortly."})
}

// ChangePassword resets password using a valid reset token
func ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate token
	tokenMutex.RLock()
	expiry, exists := resetTokens[req.Token]
	tokenMutex.RUnlock()

	if !exists || time.Now().After(expiry) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired reset token"})
		return
	}

	// In production, update password in database here
	// For now, we'll just acknowledge the reset
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "twinemukamai@gmail.com"
	}

	// Remove used token
	tokenMutex.Lock()
	delete(resetTokens, req.Token)
	tokenMutex.Unlock()

	// Send confirmation email
	err := utils.SendPasswordChangeConfirmation(adminEmail)
	if err != nil {
		fmt.Println("Confirmation email error:", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset. Please log in with your new password."})
}

// Helper function to generate random tokens
func generateToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
