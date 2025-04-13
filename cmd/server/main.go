package main

import (
	"fmt"
	"log"

	"groops/internal/handlers"

	"github.com/gin-gonic/gin"
)

// This is our main function - the entry point of our application
func main() {
	// Initialize Gin router
	router := gin.Default()

	// Configure trusted proxies
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// Basic routes
	router.GET("/", handlers.HomeHandler)
	router.GET("/health", handlers.HealthHandler)

	// Account routes
	router.POST("/accounts", handlers.CreateAccount)
	router.GET("/accounts/:username", handlers.GetAccount)

	// Group routes
	router.POST("/groups", handlers.CreateGroup)
	router.GET("/groups", handlers.GetGroups)

	// Start the server
	fmt.Println("Server starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
