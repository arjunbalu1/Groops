package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware is a Gin middleware to validate JWT tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the token from the cookie
		tokenString, err := c.Cookie(AccessTokenCookieName)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Validate the token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Check if it's an access token
		if claims.TokenType != AccessToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			c.Abort()
			return
		}

		// Store the claims in the context for later use
		c.Set("username", claims.Username)
		c.Set("claims", claims)

		c.Next()
	}
}

// GetCurrentUser extracts the username from the Gin context
func GetCurrentUser(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	return username.(string)
}
