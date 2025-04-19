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
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Configure trusted proxies
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// Basic routes
	router.GET("/", handlers.HomeHandler)
	router.GET("/health", handlers.HealthHandler)

	// Auth routes (no auth required)
	router.POST("/auth/login", handlers.Login)

	// Account routes (no auth required)
	router.POST("/accounts", handlers.CreateAccount)
	router.GET("/accounts/:username", handlers.GetAccount)

	// Public group routes
	router.GET("/public/groups", handlers.GetGroups)

	// Protected routes (auth required)
	protected := router.Group("")
	protected.Use(auth.AuthMiddleware())
	{
		// Auth routes that require authentication
		protected.POST("/auth/logout", handlers.Logout) // Logout requires auth to invalidate token
		protected.GET("/auth/me", handlers.GetCurrentUser)

		// Protected group routes
		protected.POST("/groups", handlers.CreateGroup)
		protected.GET("/groups", handlers.GetGroups)
	}

	// Start the server
	fmt.Println("Server starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
