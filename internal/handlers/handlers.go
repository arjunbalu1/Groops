package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HomeHandler handles requests to the root path "/"
func HomeHandler(c *gin.Context) {
	c.String(http.StatusOK, "Welcome to Groops!")
}

// HealthHandler is a simple health check endpoint
func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
