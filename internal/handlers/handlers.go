package handlers

import (
	"groops/internal/auth"
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

// LoginHandler redirects to Google OAuth login
func LoginHandler(c *gin.Context) {
	url, err := auth.GetLoginURL(c)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to generate login URL", err)
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
	if username == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/create-profile")
		return
	}
	c.String(http.StatusOK, "Welcome to your dashboard, %s!", username)
}

// CreateProfilePageHandler serves the profile creation page
func CreateProfilePageHandler(c *gin.Context) {
	// Show the profile creation form for new users
	email := c.GetString("email")
	name := c.GetString("name")
	picture := c.GetString("picture")
	c.String(http.StatusOK, "Create your profile. Suggested email: %s, name: %s, picture: %s", email, name, picture)
}
