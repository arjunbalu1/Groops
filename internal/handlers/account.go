package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// validatePassword checks if password meets security requirements
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hasLetter := false
	hasNumber := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		} else if unicode.IsNumber(char) {
			hasNumber = true
		}

		if hasLetter && hasNumber {
			return nil
		}
	}

	return fmt.Errorf("password must contain at least one letter and one number")
}

// CreateAccount handles new user registration
func CreateAccount(c *gin.Context) {
	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Validate password strength
	if err := validatePassword(req.Password); err != nil {
		handleError(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// Password hashing is now handled in the Account model's BeforeCreate hook (don't remove this comment)
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
		// Check for common database errors like duplicate usernames
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "username") {
				handleError(c, http.StatusConflict, "Username already exists", err)
			} else if strings.Contains(err.Error(), "email") {
				handleError(c, http.StatusConflict, "Email already in use", err)
			} else {
				handleError(c, http.StatusConflict, "Account creation failed: duplicate data", err)
			}
			return
		}

		handleError(c, http.StatusInternalServerError, "Failed to create account", err)
		return
	}

	c.JSON(http.StatusCreated, account)
}

// GetAccount retrieves account information
func GetAccount(c *gin.Context) {
	username := c.Param("username")

	db := database.GetDB()
	var account models.Account
	if err := db.Preload("Activities").Preload("OwnedGroups").Preload("JoinedGroups").
		Where("username = ?", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			handleError(c, http.StatusNotFound, "Account not found", err)
			return
		}
		handleError(c, http.StatusInternalServerError, "Failed to retrieve account", err)
		return
	}

	c.JSON(http.StatusOK, account)
}

// LoginHandler handles user authentication and JWT token generation
func LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, http.StatusBadRequest, "Invalid login request", err)
		return
	}

	// Find the account by username
	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", req.Username).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleError(c, http.StatusUnauthorized, "Invalid credentials", err)
			return
		}
		handleError(c, http.StatusInternalServerError, "Login attempt failed", err)
		return
	}

	// Verify the password
	if !account.VerifyPassword(req.Password) {
		handleError(c, http.StatusUnauthorized, "Invalid credentials",
			fmt.Errorf("password verification failed for user %s", req.Username))
		return
	}

	// Generate JWT tokens
	accessToken, accessExpiry, err := auth.GenerateToken(account.Username, auth.AccessToken)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to generate access token", err)
		return
	}

	refreshToken, refreshExpiry, err := auth.GenerateToken(account.Username, auth.RefreshToken)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to generate refresh token", err)
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
		handleError(c, http.StatusUnauthorized, "Refresh token required", err)
		return
	}

	// Validate the refresh token
	claims, err := auth.ValidateToken(refreshToken)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	}

	// Ensure it's a refresh token
	if claims.TokenType != auth.RefreshToken {
		handleError(c, http.StatusUnauthorized, "Invalid token type",
			fmt.Errorf("token type mismatch: expected refresh, got %s", claims.TokenType))
		return
	}

	// Generate a new access token
	accessToken, accessExpiry, err := auth.GenerateToken(claims.Username, auth.AccessToken)
	if err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to generate token", err)
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
