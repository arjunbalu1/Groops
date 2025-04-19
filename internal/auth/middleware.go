package auth

import (
	"groops/internal/database"
	"groops/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware verifies the JWT token from the cookie and validates token version
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from cookie
		token, err := c.Cookie("auth_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Validate the token format and signature
		claims, err := ValidateToken(token)
		if err != nil {
			if err.Error() == "token has expired" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			}
			c.Abort()
			return
		}

		// Get user from database to verify token version
		db := database.GetDB()
		var account models.Account
		if err := db.Where("username = ?", claims.Username).First(&account).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// Verify token version
		if claims.TokenVersion != account.TokenVersion {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			c.Abort()
			return
		}

		// Set username in the context for handlers to use
		c.Set("username", claims.Username)
		c.Next()
	}
}
