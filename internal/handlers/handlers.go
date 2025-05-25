package handlers

import (
	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/models"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HomeHandler handles requests to the root path "/"
func HomeHandler(c *gin.Context) {
	c.String(http.StatusOK, "welcome to groops")
}

// HealthHandler is a simple health check endpoint
func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// LoginHandler redirects to Google OAuth login
func LoginHandler(c *gin.Context) {
	url, err := auth.GetLoginURL(c)
	if err != nil {
		log.Printf("Error: Failed to generate login URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate login URL"})
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallbackHandler processes the OAuth callback from Google
func GoogleCallbackHandler(c *gin.Context) {
	auth.HandleGoogleCallback(c)
}

// LogoutHandler handles user logout
func LogoutHandler(c *gin.Context) {
	auth.LogoutHandler(c)
}

// DashboardHandler serves the user dashboard page
func DashboardHandler(c *gin.Context) {
	username := c.GetString("username")
	if username == "" || strings.HasPrefix(username, "temp-") {
		c.Redirect(http.StatusTemporaryRedirect, "/create-profile")
		return
	}
	c.String(http.StatusOK, "Welcome to your dashboard, %s!", username)
}

// CreateProfilePageHandler serves the profile creation page
func CreateProfilePageHandler(c *gin.Context) {
	// Check if user already has a non-temporary profile
	username := c.GetString("username")
	if username != "" && !strings.HasPrefix(username, "temp-") {
		// User already has a permanent profile, redirect to home
		c.Redirect(http.StatusTemporaryRedirect, "https://groops.fun")
		return
	}

	// Return user info for the React app to use
	c.JSON(http.StatusOK, gin.H{
		"needsProfile": true,
		"email":        c.GetString("email"),
		"name":         c.GetString("name"),
		"picture":      c.GetString("picture"),
	})
}

// GetStats returns platform statistics
func GetStats(c *gin.Context) {
	db := database.DB

	var userCount int64
	var groupCount int64

	// Count total users
	if err := db.Model(&models.Account{}).Count(&userCount).Error; err != nil {
		log.Printf("Error counting users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch statistics"})
		return
	}

	// Count total groups
	if err := db.Model(&models.Group{}).Count(&groupCount).Error; err != nil {
		log.Printf("Error counting groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  userCount,
		"groups": groupCount,
	})
}
