package handlers

import (
	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LoginRequest represents the data needed for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user authentication and issues a JWT token
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the account
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

	// TODO: Implement proper password verification
	// For now, we're comparing unhashed passwords for development
	if account.HashedPass != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Update last login time
	if err := db.Model(&account).Update("last_login", time.Now()).Error; err != nil {
		// Log the error but don't fail the login
		// In a production environment, consider adding proper error logging
	}

	// Set auth cookie with current token version
	if err := auth.SetAuthCookie(c, account.Username, account.TokenVersion); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"user": gin.H{
			"username": account.Username,
			"email":    account.Email,
		},
	})
}

// Logout handles user logout by invalidating the token and clearing cookie
func Logout(c *gin.Context) {
	username := auth.GetUsernameFromContext(c)

	// If there's a valid user in the context, invalidate their token
	if username != "" {
		db := database.GetDB()

		// Increment the token version to invalidate all existing tokens
		result := db.Model(&models.Account{}).
			Where("username = ?", username).
			Update("token_version", gorm.Expr("token_version + 1"))

		if result.Error != nil {
			// Log the error but continue with logout
			// In production, consider proper error handling
		}
	}

	// Clear the auth cookie
	auth.ClearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logout successful"})
}

// GetCurrentUser returns the currently authenticated user
func GetCurrentUser(c *gin.Context) {
	username := auth.GetUsernameFromContext(c)
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":   account.Username,
		"email":      account.Email,
		"dateJoined": account.DateJoined,
		"rating":     account.Rating,
		"lastLogin":  account.LastLogin,
	})
}
