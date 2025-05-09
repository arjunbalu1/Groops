package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRealClientIP extracts the real client IP from the request headers
// It prioritizes X-Real-IP, then X-Forwarded-For, and finally falls back to c.ClientIP()
func GetRealClientIP(c *gin.Context) string {
	// Try X-Real-IP header first (most reliable)
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	// Try X-Forwarded-For header next
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to Gin's built-in method
	return c.ClientIP()
}
