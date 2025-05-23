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

	// Check if c.Request is available
	if c.Request != nil {
		// Try to get IP from RemoteAddr directly
		if c.Request.RemoteAddr != "" {
			// RemoteAddr is typically in the format "IP:port"
			ip := strings.Split(c.Request.RemoteAddr, ":")[0]
			return ip
		}
	}

	// Try ClientIP as a last resort, with panic recovery
	defer func() {
		if r := recover(); r != nil {
			// If we panic here, just return a fallback IP
		}
	}()

	return c.ClientIP()
}
