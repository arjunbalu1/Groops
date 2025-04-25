package main

import (
	"fmt"
	"log"
	"os"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from project root
	if err := godotenv.Load("../../.env"); err != nil {
		// Try standard location as fallback
		if err := godotenv.Load(); err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

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
		api.GET("/accounts/:username/history", handlers.GetAccountEventHistory)
		api.PUT("/accounts/:username", handlers.UpdateAccount)

		// Group routes
		api.POST("/groups", handlers.CreateGroup)
		api.GET("/groups", handlers.GetGroups)
		api.GET("/groups/:group_id", handlers.GetGroupByID)
		api.POST("/groups/:group_id/join", handlers.JoinGroup)
		api.POST("/groups/:group_id/leave", handlers.LeaveGroup)

		// New endpoints for organiser actions
		api.GET("/groups/:group_id/pending-members", handlers.ListPendingMembers)
		api.POST("/groups/:group_id/members/:username/approve", handlers.ApproveJoinRequest)
		api.POST("/groups/:group_id/members/:username/reject", handlers.RejectJoinRequest)

		// Notification routes
		api.GET("/notifications", handlers.ListNotifications)
		api.GET("/notifications/unread-count", handlers.GetUnreadNotificationCount)
	}

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s...\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
