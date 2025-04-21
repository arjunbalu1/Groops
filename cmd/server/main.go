package main

import (
	"fmt"
	"log"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/handlers"

	"github.com/gin-gonic/gin"
)

// This is our main function - the entry point of our application
func main() {
	// Initialize database
	database.InitDB()

	// Initialize Gin router
	router := gin.Default()

	// Configure trusted proxies
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// Public routes
	router.GET("/", handlers.HomeHandler)
	router.GET("/health", handlers.HealthHandler)

	// Authentication routes
	router.POST("/accounts", handlers.CreateAccount)
	router.POST("/auth/login", handlers.LoginHandler)
	router.POST("/auth/refresh", handlers.RefreshTokenHandler)

	// Protected API routes
	api := router.Group("/api")
	api.Use(auth.AuthMiddleware())
	{
		// Account routes
		api.GET("/accounts/:username", handlers.GetAccount)

		// Group routes
		api.POST("/groups", handlers.CreateGroup)
		api.GET("/groups", handlers.GetGroups)
	}

	// Start the server
	fmt.Println("Server starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
