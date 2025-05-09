package main

import (
	"fmt"
	"log"
	"os"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/handlers"
	"groops/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from project root
	if err := godotenv.Load("../../.env"); err != nil {
		// Try standard location as fallback
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: .env file not found, relying on environment variables")
		}
	}

	// Initialize Google OAuth
	if err := auth.InitOAuth(); err != nil {
		log.Fatalf("Failed to initialize Google OAuth: %v", err)
	}

	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Google Maps client
	if err := services.InitMapsClient(); err != nil {
		log.Printf("Warning: Failed to initialize Google Maps client: %v", err)
		// Continue anyway - not critical for app startup
	}

	// Initialize and start the event reminder worker
	reminderWorker := services.NewReminderWorker()
	reminderWorker.Start()
	log.Println("Event reminder worker started")

	// Set Gin mode based on environment
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	router := gin.Default()

	// Load HTML templates
	router.LoadHTMLGlob("internal/templates/*.html")

	// Configure trusted proxies
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// CORS Middleware Configuration
	// remove and test later if cross origin request fail
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	// Public routes
	router.GET("/", handlers.HomeHandler)
	router.GET("/health", handlers.HealthHandler)

	// Auth routes
	router.GET("/auth/login", handlers.LoginHandler)
	router.GET("/auth/google/callback", handlers.GoogleCallbackHandler)
	router.GET("/auth/logout", handlers.LogoutHandler)

	authPageGroup := router.Group("/")
	authPageGroup.Use(auth.AuthMiddleware())
	{
		// Account creation page - requires authentication but not a full user profile
		authPageGroup.GET("/create-profile", handlers.CreateProfilePageHandler)
		authPageGroup.GET("/dashboard", handlers.DashboardHandler)
		authPageGroup.POST("/api/profile/register", handlers.CreateProfile)
	}

	// Protected API routes - require authentication with a full user profile
	api := router.Group("/api")
	api.Use(auth.AuthMiddleware(), auth.RequireFullProfileMiddleware())
	{
		// Account routes
		api.GET("/accounts/:username", handlers.GetAccount)
		api.GET("/accounts/:username/history", handlers.GetAccountEventHistory)
		api.PUT("/profile", handlers.UpdateAccount)

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
		api.POST("/groups/:group_id/members/:username/remove", handlers.RemoveMember)

		// Notification routes
		api.GET("/notifications", handlers.ListNotifications)
		api.GET("/notifications/unread-count", handlers.GetUnreadNotificationCount)

		// Location validation route
		api.GET("/locations/validate", handlers.ValidateLocation)
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
