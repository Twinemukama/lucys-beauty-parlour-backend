package middleware

import (
    "net/http"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
)

func AdminAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        if auth == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
            return
        }
        parts := strings.Fields(auth)
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
            return
        }
        tokenStr := parts[1]
        secret := os.Getenv("JWT_SECRET")
        if secret == "" {
            secret = "secret"
        }
        token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
            if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
                return nil, jwt.ErrTokenUnverifiable
            }
            return []byte(secret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }
        c.Next()
    }
}
