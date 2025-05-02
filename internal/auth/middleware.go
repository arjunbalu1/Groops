package auth

import (
	"context"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
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
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile", "openid"},
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
	return googleOAuthConfig.AuthCodeURL(state,
		oauth2.SetAuthURLParam("prompt", "select_account"),
	), nil
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

	// Extract ID token from the token response
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get id_token"})
		c.Abort()
		return
	}

	// Verify the ID token
	payload, err := verifyIDToken(rawIDToken, googleOAuthConfig.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to verify id_token: %v", err)})
		c.Abort()
		return
	}

	// Extract user info from the verified payload
	userInfo, err := extractUserInfoFromPayload(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to extract user info from token"})
		c.Abort()
		return
	}

	// Check if user already exists
	var existingAccount models.Account
	db := database.GetDB()
	if err := db.Where("google_id = ?", userInfo.Sub).First(&existingAccount).Error; err == nil {
		// User exists, create session with username
		if err := CreateSession(c, userInfo, existingAccount.Username); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			c.Abort()
			return
		}

		// Redirect to dashboard or home page
		c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
		return
	}

	// User does not exist
	// Generate a temporary random username
	randomID, err := GenerateRandomString(8)
	if err != nil {
		fmt.Printf("Warning: Failed to generate temporary username: %v\n", err)
		randomID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	tempUsername := fmt.Sprintf("temp-%s", randomID)

	// Create a temporary account record
	tempAccount := models.Account{
		GoogleID:      userInfo.Sub,
		Username:      tempUsername,
		Email:         userInfo.Email,
		EmailVerified: userInfo.EmailVerified,
		FullName:      userInfo.Name,
		GivenName:     userInfo.GivenName,
		FamilyName:    userInfo.FamilyName,
		Locale:        userInfo.Locale,
		DateJoined:    time.Now(),
		LastLogin:     time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Rating:        5.0,
		AvatarURL:     userInfo.Picture,
	}

	// Create the account
	if err := db.Create(&tempAccount).Error; err != nil {
		fmt.Printf("Warning: Failed to create temporary account: %v\n", err)
	}

	// Create session with temporary username
	if err := CreateSession(c, userInfo, tempUsername); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		c.Abort()
		return
	}

	// Redirect to profile creation page
	c.Redirect(http.StatusTemporaryRedirect, "/create-profile")
}

// verifyIDToken verifies the ID token using Google's official library
func verifyIDToken(idToken string, audience string) (*idtoken.Payload, error) {
	// Use Google's idtoken library to verify the token
	payload, err := idtoken.Validate(context.Background(), idToken, audience)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}
	return payload, nil
}

// extractUserInfoFromPayload extracts user info from the verified token payload
func extractUserInfoFromPayload(payload *idtoken.Payload) (*UserInfo, error) {
	userInfo := &UserInfo{
		Sub:   payload.Subject,
		Email: payload.Claims["email"].(string),
	}

	// Extract other fields if they exist
	if name, ok := payload.Claims["name"].(string); ok {
		userInfo.Name = name
	}
	if picture, ok := payload.Claims["picture"].(string); ok {
		userInfo.Picture = picture
	}
	if given_name, ok := payload.Claims["given_name"].(string); ok {
		userInfo.GivenName = given_name
	}
	if family_name, ok := payload.Claims["family_name"].(string); ok {
		userInfo.FamilyName = family_name
	}
	if locale, ok := payload.Claims["locale"].(string); ok {
		userInfo.Locale = locale
	}
	if email_verified, ok := payload.Claims["email_verified"].(bool); ok {
		userInfo.EmailVerified = email_verified
	}

	return userInfo, nil
}

// AuthMiddleware validates the session
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the session from the request
		session, err := GetSession(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Verify the session hasn't expired
		if session.IsExpired() {
			DeleteSession(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired, please log in again"})
			c.Abort()
			return
		}

		// Store user info in context for handlers to use
	    // If session has a username, set it in the context
		if session.Username != "" {
			c.Set("username", session.Username)
		}
		c.Set("sub", session.UserID)
		c.Set("email", session.Email)
		c.Set("name", session.Name)
		c.Set("picture", session.Picture)
		c.Set("email_verified", session.EmailVerified)
		c.Set("given_name", session.GivenName)
		c.Set("family_name", session.FamilyName)
		c.Set("locale", session.Locale)

		c.Next()
	}
}

// LogoutHandler handles user logout
func LogoutHandler(c *gin.Context) {
	DeleteSession(c)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
