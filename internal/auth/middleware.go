package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
)

// InitOAuth initializes the Google OAuth configuration
func InitOAuth() error {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL must be set")
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	return nil
}

// GetLoginURL returns the Google OAuth login URL with a secure state parameter
func GetLoginURL(c *gin.Context) (string, error) {
	// Generate and store a secure random state
	state, err := SetOAuthState(c)
	if err != nil {
		return "", err
	}

	// Generate the authorization URL with the state parameter
	return googleOAuthConfig.AuthCodeURL(state), nil
}

// HandleGoogleCallback processes the OAuth callback from Google
func HandleGoogleCallback(c *gin.Context) {
	// Verify state parameter (CSRF protection)
	state := c.Query("state")
	if !VerifyOAuthState(c, state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state, possible CSRF attack"})
		c.Abort()
		return
	}

	// Exchange auth code for token
	code := c.Query("code")
	token, err := googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "code exchange failed"})
		c.Abort()
		return
	}

	// Get user info from Google
	userInfo, err := getUserInfo(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		c.Abort()
		return
	}

	// Check if user already exists
	var existingAccount models.Account
	if err := database.GetDB().Where("google_id = ?", userInfo.Sub).First(&existingAccount).Error; err == nil {
		// User exists, create session
		if err := CreateSession(c, token, userInfo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			c.Abort()
			return
		}

		// Redirect to dashboard or home page
		c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
		return
	}

	// User does not exist, create session without username
	if err := CreateSession(c, token, userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		c.Abort()
		return
	}

	// Redirect to profile creation page
	c.Redirect(http.StatusTemporaryRedirect, "/create-profile")
}

// getUserInfo gets the user info from the Google API
func getUserInfo(token *oauth2.Token) (*UserInfo, error) {
	// Create a client that uses the token
	client := googleOAuthConfig.Client(context.Background(), token)

	// Make a request to the userinfo endpoint
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

// AuthMiddleware validates Google JWT tokens and refreshes them if needed
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the session from the request
		session, err := GetSession(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Refresh the token if needed
		if session.NeedsTokenRefresh() {
			if err := RefreshSessionToken(c, session); err != nil {
				// Token refresh failed, force re-login
				DeleteSession(c)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired, please log in again"})
				c.Abort()
				return
			}
		}

		// Get user info from Google to verify the token is still valid
		client := googleOAuthConfig.Client(context.Background(), &oauth2.Token{
			AccessToken: session.AccessToken,
			TokenType:   "Bearer",
			Expiry:      session.TokenExpiry,
		})

		resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil || resp.StatusCode != http.StatusOK {
			// Token is invalid
			DeleteSession(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token, please log in again"})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		// Parse user info
		var userInfo UserInfo
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user info"})
			c.Abort()
			return
		}

		// Store user info in context for handlers to use
		c.Set("sub", userInfo.Sub)
		c.Set("email", userInfo.Email)
		c.Set("name", userInfo.Name)
		c.Set("picture", userInfo.Picture)

		// If session has a username, set it in the context
		if session.Username != "" {
			c.Set("username", session.Username)
		}

		c.Next()
	}
}

// LogoutHandler handles user logout
func LogoutHandler(c *gin.Context) {
	DeleteSession(c)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
