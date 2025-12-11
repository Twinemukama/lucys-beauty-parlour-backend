package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"lucys-beauty-parlour-backend/storage"
	"lucys-beauty-parlour-backend/utils"

	"github.com/gin-gonic/gin"
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

var RefreshDB *storage.RefreshStore

func AdminLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if req.Email != adminEmail || req.Password != adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Access token (15 min)
	access, err := utils.GenerateAccessToken(req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create token"})
		return
	}

	// Refresh token (7 days)
	refresh, err := utils.GenerateRefreshToken()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create token"})
		return
	}

	// Store refresh token server-side
	RefreshDB.Save(refresh)

	// Send refresh token as HttpOnly cookie
	c.SetCookie("refresh_token", refresh, 7*24*3600, "/", "", false, true)

	c.JSON(200, gin.H{
		"access_token": access,
	})
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

func RefreshToken(c *gin.Context) {
	refreshCookie, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(401, gin.H{"error": "no refresh token"})
		return
	}

	// Check if refresh token exists in store
	if !RefreshDB.Exists(refreshCookie) {
		c.JSON(401, gin.H{"error": "invalid refresh token"})
		return
	}

	token, err := utils.VerifyRefreshToken(refreshCookie)
	if err != nil || !token.Valid {
		c.JSON(401, gin.H{"error": "invalid refresh token"})
		return
	}

	// Create new access token
	access, err := utils.GenerateAccessToken("admin")
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate access token"})
		return
	}

	c.JSON(200, gin.H{"access_token": access})
}

func Logout(c *gin.Context) {
	refreshCookie, err := c.Cookie("refresh_token")
	if err == nil {
		RefreshDB.Delete(refreshCookie)
	}

	// Clear cookie
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(200, gin.H{"message": "logged out"})
}
