package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"groops/internal/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
func CreateSession(c *gin.Context, userInfo *UserInfo, username ...string) error {
	// Generate a random session ID
	sessionID, err := GenerateRandomString(SessionIDLength)
	if err != nil {
		return fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Get database connection
	db := database.GetDB()

	// Get real client IP using the utility function
	clientIP := utils.GetRealClientIP(c)

	// Create a new session with user info
	session := models.Session{
		ID:            sessionID,
		UserID:        userInfo.Sub,
		Username:      "",
		Email:         userInfo.Email,
		Name:          userInfo.Name,
		Picture:       userInfo.Picture,
		EmailVerified: userInfo.EmailVerified,
		GivenName:     userInfo.GivenName,
		FamilyName:    userInfo.FamilyName,
		Locale:        userInfo.Locale,
		IPAddress:     clientIP,
		UserAgent:     c.Request.UserAgent(),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(models.SessionDuration),
	}

	// Set username and check if it's a temporary account
	isTemp := strings.HasPrefix(username[0], "temp-")
	session.Username = username[0]

	// Store the session in the database
	if err := db.Create(&session).Error; err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Log login activity
	loginLog := models.LoginLog{
		Username:  session.Username,
		GoogleID:  userInfo.Sub,
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		LoginTime: time.Now(),
		IPAddress: clientIP,
		UserAgent: c.Request.UserAgent(),
		SessionID: sessionID,
		IsTemp:    isTemp,
	}

	if err := db.Create(&loginLog).Error; err != nil {
		// Just log the error, don't fail the login process
		fmt.Printf("Warning: Failed to create login log: %v\n", err)
	}

	// Set the session cookie with SameSite=Strict
	secure := gin.Mode() != gin.DebugMode

	// Create cookie with SameSite=Strict
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Domain:   "",
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	// Set the cookie in the response
	http.SetCookie(c.Writer, cookie)

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

// DeleteSession removes the session and clears cookies
func DeleteSession(c *gin.Context) {
	// Get the session ID
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		// Get database connection
		db := database.GetDB()

		// Update login log with logout time
		now := time.Now()
		if err := db.Model(&models.LoginLog{}).
			Where("session_id = ?", sessionID).
			Update("logout_time", now).Error; err != nil {
			// Just log the error, continue with session deletion
			fmt.Printf("Warning: Failed to update login log with logout time: %v\n", err)
		}

		// Delete from database
		db.Where("id = ?", sessionID).Delete(&models.Session{})
	}

	// Clear the session cookie with the same secure setting as creation
	secure := gin.Mode() != gin.DebugMode

	// Create an expired cookie with SameSite=Strict
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   "",
		MaxAge:   -1,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	// Set the cookie in the response
	http.SetCookie(c.Writer, cookie)
}

// SetOAuthState generates and stores a random state for CSRF protection
func SetOAuthState(c *gin.Context) (string, error) {
	state, err := GenerateRandomString(StateLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state in a temporary cookie with SameSite=Lax
	// This cookie is only used during the OAuth flow and will be cleared after
	// We use Lax instead of Strict to allow redirects from Google OAuth
	secure := gin.Mode() != gin.DebugMode

	// Create cookie with SameSite=Lax for OAuth flow
	cookie := &http.Cookie{
		Name:     StateCookieName,
		Value:    state,
		Path:     "/",
		Domain:   "",
		MaxAge:   int(10 * time.Minute.Seconds()), // 10 minutes expiry
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode, // Lax allows cookies on redirects from OAuth
	}

	// Set the cookie in the response
	http.SetCookie(c.Writer, cookie)

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
	secure := gin.Mode() != gin.DebugMode
	// Create an expired cookie with SameSite=Lax to match the setting used when creating it
	cookie := &http.Cookie{
		Name:     StateCookieName,
		Value:    "",
		Path:     "/",
		Domain:   "",
		MaxAge:   -1,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode, // Match the setting used when creating
	}

	// Set the cookie in the response
	http.SetCookie(c.Writer, cookie)

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
