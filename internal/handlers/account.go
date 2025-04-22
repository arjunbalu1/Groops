package handlers

import (
	"net/http"
	"time"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAccount handles new user registration
func CreateAccount(c *gin.Context) {
	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Password hashing is now handled in the Account model's BeforeCreate hook
	now := time.Now()
	account := models.Account{
		Username:   req.Username,
		Email:      req.Email,
		HashedPass: req.Password,
		DateJoined: now,
		Rating:     5.0,
		LastLogin:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	db := database.GetDB()
	if err := db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
}

// LogActivity adds a new activity to user's history
func LogActivity(username string, eventType string, groupID string) error {
	activity := models.ActivityLog{
		Username:  username,
		EventType: eventType,
		GroupID:   groupID,
		Timestamp: time.Now(),
	}

	db := database.GetDB()
	return db.Create(&activity).Error
}

// GetAccount retrieves account information
func GetAccount(c *gin.Context) {
	username := c.Param("username")

	db := database.GetDB()
	var account models.Account
	if err := db.Preload("Activities").Preload("OwnedGroups").Preload("JoinedGroups").
		Where("username = ?", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

// LoginHandler handles user authentication and JWT token generation
func LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the account by username
	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", req.Username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Verify the password
	if !account.VerifyPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Generate JWT tokens
	accessToken, accessExpiry, err := auth.GenerateToken(account.Username, auth.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refreshToken, refreshExpiry, err := auth.GenerateToken(account.Username, auth.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Set SameSite mode to Strict for all cookies
	c.SetSameSite(http.SameSiteStrictMode)

	// Set secure HttpOnly cookies
	// Access token cookie
	c.SetCookie(
		auth.AccessTokenCookieName,
		accessToken,
		int(auth.AccessTokenExpiry.Seconds()),
		"/api", // Only sent to API routes
		"",     // Domain - blank for current domain
		true,   // Secure - HTTPS only
		true,   // HttpOnly - not accessible via JavaScript
	)

	// Refresh token cookie
	c.SetCookie(
		auth.RefreshTokenCookieName,
		refreshToken,
		int(auth.RefreshTokenExpiry.Seconds()),
		"/auth/refresh", // Only sent to refresh endpoint
		"",              // Domain
		true,            // Secure
		true,            // HttpOnly
	)

	// Update last login time
	db.Model(&account).Update("last_login", time.Now())

	// Return success to the client (without tokens in the body)
	c.JSON(http.StatusOK, gin.H{
		"username":              account.Username,
		"access_token_expires":  accessExpiry,
		"refresh_token_expires": refreshExpiry,
	})
}

// RefreshTokenHandler handles token refresh requests
func RefreshTokenHandler(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie(auth.RefreshTokenCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token required"})
		return
	}

	// Validate the refresh token
	claims, err := auth.ValidateToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	// Ensure it's a refresh token
	if claims.TokenType != auth.RefreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
		return
	}

	// Generate a new access token
	accessToken, accessExpiry, err := auth.GenerateToken(claims.Username, auth.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Set SameSite mode to Strict
	c.SetSameSite(http.SameSiteStrictMode)

	// Set new access token cookie
	c.SetCookie(
		auth.AccessTokenCookieName,
		accessToken,
		int(auth.AccessTokenExpiry.Seconds()),
		"/api", // Only sent to API routes
		"",     // Domain
		true,   // Secure
		true,   // HttpOnly
	)

	// Return the expiry information (not the token itself)
	c.JSON(http.StatusOK, gin.H{
		"username":             claims.Username,
		"access_token_expires": accessExpiry,
	})
}
