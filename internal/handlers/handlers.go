package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleError provides a consistent way to handle and log errors
func handleError(c *gin.Context, status int, message string, err error) {
	log.Printf("Error: %v", err)
	c.JSON(status, gin.H{"error": message})
}

// HomeHandler handles requests to the root path "/"
func HomeHandler(c *gin.Context) {
	c.String(http.StatusOK, "Welcome to Groops!")
}

// HealthHandler is a simple health check endpoint
func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
