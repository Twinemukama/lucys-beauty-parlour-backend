package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"lucys-beauty-parlour-backend/database"
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

type resetTokenEntry struct {
	Email     string
	ExpiresAt time.Time
}

// In-memory storage for password reset tokens.
var (
	resetTokens = make(map[string]resetTokenEntry)
	tokenMutex  sync.RWMutex
)

var RefreshDB *storage.RefreshStore
var AdminDB *sql.DB

func AdminLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.TrimSpace(req.Email)
	password := req.Password

	if AdminDB != nil {
		ok, err := database.ValidateAdminCredentials(AdminDB, email, password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
			return
		}
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
	} else {
		adminEmail := os.Getenv("ADMIN_EMAIL")
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if email != adminEmail || password != adminPassword {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
	}

	// Access token (15 min)
	access, err := utils.GenerateAccessToken(email)
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

// ForgotPassword sends a password reset email.
func ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.TrimSpace(req.Email)

	if AdminDB != nil {
		exists, err := database.AdminExists(AdminDB, email)
		if err != nil || !exists {
			// Do not reveal account existence.
			c.JSON(http.StatusOK, gin.H{"message": "Check your admin email for the password reset link."})
			return
		}
	} else {
		adminEmail := os.Getenv("ADMIN_EMAIL")
		if email != adminEmail {
			c.JSON(http.StatusOK, gin.H{"message": "Check your admin email for the password reset link."})
			return
		}
	}

	// Generate reset token
	resetToken := generateToken(32)

	// Store token with 1-hour expiry
	tokenMutex.Lock()
	resetTokens[resetToken] = resetTokenEntry{Email: email, ExpiresAt: time.Now().Add(1 * time.Hour)}
	tokenMutex.Unlock()

	// Send email
	err := utils.SendPasswordResetEmail(email, resetToken)
	if err != nil {
		// Log error but return success to prevent email enumeration
		fmt.Println("Email send error:", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "If this email exists, you will receive a password reset link shortly."})
}

// ChangePassword resets password using a valid reset token.
func ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.NewPassword) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_password is required"})
		return
	}

	// Validate token
	tokenMutex.RLock()
	entry, exists := resetTokens[req.Token]
	tokenMutex.RUnlock()

	if !exists || time.Now().After(entry.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired reset token"})
		return
	}

	adminEmail := entry.Email
	if adminEmail == "" {
		adminEmail = os.Getenv("ADMIN_EMAIL")
	}

	if AdminDB != nil {
		if err := database.UpdateAdminPassword(AdminDB, adminEmail, req.NewPassword); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "admin account not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
			return
		}
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

// Helper function to generate random tokens.
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
