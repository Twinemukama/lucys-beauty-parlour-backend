package handlers

import (
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
)

type loginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

func AdminLogin(c *gin.Context) {
    var req loginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    adminUser := os.Getenv("ADMIN_USER")
    adminPass := os.Getenv("ADMIN_PASS")
    if adminUser == "" {
        adminUser = "admin"
    }
    if adminPass == "" {
        adminPass = "password"
    }

    if req.Username != adminUser || req.Password != adminPass {
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
