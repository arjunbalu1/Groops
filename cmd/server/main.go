package main

import (
	"fmt"
	"log"
	"os"

	"groops/internal/handlers"

	"github.com/gin-gonic/gin"
)

// This is our main function - the entry point of our application
func main() {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.Default()

	// Configure trusted proxies
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// Define our routes
	router.GET("/", handlers.HomeHandler)
	router.GET("/health", handlers.HealthHandler)

	// Start the server
	fmt.Println("Server starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
