package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	// SessionCookieName is the name of the cookie that stores the session ID
	SessionCookieName = "groops_session"
	// StateCookieName is the name of the cookie that temporarily stores the OAuth state
	StateCookieName = "groops_oauth_state"
	// SessionIDLength is the length of the random session ID in bytes
	SessionIDLength = 32
	// StateLength is the length of the random state string in bytes
	StateLength = 32
)

// GenerateRandomString creates a cryptographically secure random string
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// CreateSession creates a new session for the user
func CreateSession(c *gin.Context, token *oauth2.Token, userInfo *UserInfo) error {
	// Generate a random session ID
	sessionID, err := GenerateRandomString(SessionIDLength)
	if err != nil {
		return fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Create a new session
	session := models.Session{
		ID:           sessionID,
		UserID:       userInfo.Sub,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour * 24 * 30), // 30 days
	}

	// Store the session in the database
	db := database.GetDB()
	if err := db.Create(&session).Error; err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Set the session cookie
	secure := gin.Mode() != gin.DebugMode
	c.SetCookie(
		SessionCookieName,
		sessionID,
		int(session.ExpiresAt.Sub(time.Now()).Seconds()),
		"/",
		"",
		secure,
		true, // HttpOnly for security
	)

	return nil
}

// GetSession retrieves the current session from the request
func GetSession(c *gin.Context) (*models.Session, error) {
	// Get the session ID from the cookie
	sessionID, err := c.Cookie(SessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("session cookie not found: %w", err)
	}

	// Get the session from the database
	db := database.GetDB()
	var session models.Session
	if err := db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to retrieve session: %w", err)
	}

	// Check if the session has expired
	if session.IsExpired() {
		DeleteSession(c)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// RefreshSessionToken refreshes the OAuth token for the session
func RefreshSessionToken(c *gin.Context, session *models.Session) error {
	// Only refresh if needed
	if !session.NeedsTokenRefresh() {
		return nil
	}

	// Create a token source with the refresh token
	token := &oauth2.Token{
		RefreshToken: session.RefreshToken,
	}

	// Use the token source to get a new token
	newToken, err := googleOAuthConfig.TokenSource(c, token).Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update the session with the new token
	db := database.GetDB()
	updates := map[string]interface{}{
		"access_token": newToken.AccessToken,
		"token_expiry": newToken.Expiry,
	}

	// If we got a new refresh token, update that too
	if newToken.RefreshToken != "" {
		updates["refresh_token"] = newToken.RefreshToken
	}

	if err := db.Model(&session).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteSession removes the session and clears cookies
func DeleteSession(c *gin.Context) {
	// Get the session ID
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		// Delete from database
		db := database.GetDB()
		db.Where("id = ?", sessionID).Delete(&models.Session{})
	}

	// Clear the session cookie
	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
}

// SetOAuthState generates and stores a random state for CSRF protection
func SetOAuthState(c *gin.Context) (string, error) {
	state, err := GenerateRandomString(StateLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state in a temporary cookie
	// This cookie is only used during the OAuth flow and will be cleared after
	secure := gin.Mode() != gin.DebugMode
	c.SetCookie(
		StateCookieName,
		state,
		int(10*time.Minute.Seconds()), // 10 minutes expiry
		"/",
		"",
		secure,
		true, // HttpOnly for security
	)

	return state, nil
}

// VerifyOAuthState verifies the state parameter from the OAuth callback
func VerifyOAuthState(c *gin.Context, receivedState string) bool {
	// Get the state from the cookie
	savedState, err := c.Cookie(StateCookieName)
	if err != nil {
		return false
	}

	// Clear the state cookie regardless of outcome
	c.SetCookie(StateCookieName, "", -1, "/", "", false, true)

	// Verify the state
	return savedState == receivedState
}

// LinkSessionToUser links a session to a registered user
func LinkSessionToUser(sessionID, username string) error {
	db := database.GetDB()
	return db.Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("username", username).Error
}
