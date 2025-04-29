package handlers

import (
	"net/http"
	"strconv"
	"time"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/models"

	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

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

// CreateProfile handles new Google OAuth user profile registration
func CreateProfile(c *gin.Context) {
	sub := c.GetString("sub")
	email := c.GetString("email")
	picture := c.GetString("picture") // Get Google profile picture

	if sub == "" {
		handleError(c, http.StatusBadRequest, "Missing Google ID in token", nil)
		return
	}

	if email == "" {
		handleError(c, http.StatusBadRequest, "Missing email in token", nil)
		return
	}

	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Get the session
	sessionID, err := c.Cookie(auth.SessionCookieName)
	if err != nil {
		handleError(c, http.StatusUnauthorized, "No active session", err)
		return
	}

	db := database.GetDB()

	// Check if username is already taken by someone else
	var existingUsername models.Account
	if err := db.Where("username = ? AND google_id != ?", req.Username, sub).First(&existingUsername).Error; err == nil {
		handleError(c, http.StatusConflict, "Username already taken", nil)
		return
	}

	// Check if we have a temporary account for this Google ID
	var tempAccount models.Account
	accountExists := false
	isTemp := false

	if err := db.Where("google_id = ?", sub).First(&tempAccount).Error; err == nil {
		accountExists = true
		isTemp = tempAccount.Username[:5] == "temp-" // Check if it's a temporary account
	}

	// If avatar URL is not provided, use Google profile picture
	avatarURL := req.AvatarURL
	if avatarURL == "" {
		avatarURL = picture
	}

	now := time.Now()

	if accountExists && isTemp {
		// Update the temporary account with the chosen username and other details
		updates := map[string]interface{}{
			"username":   req.Username,
			"bio":        req.Bio,
			"avatar_url": avatarURL,
			"updated_at": now,
		}

		if err := db.Model(&tempAccount).Updates(updates).Error; err != nil {
			handleError(c, http.StatusInternalServerError, "Failed to update account", err)
			return
		}

		// Retrieve the updated account
		if err := db.Where("google_id = ?", sub).First(&tempAccount).Error; err != nil {
			handleError(c, http.StatusInternalServerError, "Failed to retrieve updated account", err)
			return
		}

		// Link the session to the user
		if err := auth.LinkSessionToUser(sessionID, req.Username); err != nil {
			// Non-fatal error - log but don't fail the request
			log.Printf("Warning: Failed to link session to user: %v", err)
		}

		c.JSON(http.StatusCreated, tempAccount)
		return
	} else if accountExists && !isTemp {
		// Account exists but is not temporary - this is a conflict
		handleError(c, http.StatusConflict, "Profile already exists for this user", nil)
		return
	}

	// If we get here, we need to create a new account (should rarely happen
	// since we create temp accounts during OAuth)
	account := models.Account{
		GoogleID:   sub,
		Username:   req.Username,
		Email:      email,
		DateJoined: now,
		Rating:     5.0,
		LastLogin:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
		Bio:        req.Bio,
		AvatarURL:  avatarURL,
	}

	if err := db.Create(&account).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to create profile", err)
		return
	}

	// Link the session to the new user
	if err := auth.LinkSessionToUser(sessionID, req.Username); err != nil {
		// Non-fatal error - log but don't fail the request
		log.Printf("Warning: Failed to link session to user: %v", err)
	}

	// Get session and check for refresh token
	var session models.Session
	if err := db.Where("id = ?", sessionID).First(&session).Error; err == nil {
		// Get user's session
		// Retrieve active token from database and save it to the account if it exists
		userSession, err := auth.GetSession(c)
		if err == nil && userSession.AccessToken != "" {
			// Create a token object to pass to SaveRefreshTokenToAccount
			token := &oauth2.Token{
				AccessToken: userSession.AccessToken,
				TokenType:   "Bearer",
				Expiry:      userSession.TokenExpiry,
			}

			// Try to save the token to the account
			if err := auth.SaveRefreshTokenToAccount(db, sub, token); err != nil {
				// Non-fatal, just log it
				log.Printf("Warning: Failed to save token to account: %v", err)
			}
		}
	}

	c.JSON(http.StatusCreated, account)
}

// UpdateAccount allows a user to update their profile (bio, avatar_url)
func UpdateAccount(c *gin.Context) {
	username := c.Param("username")
	requester := c.GetString("username")

	// Only the user themselves can update their profile
	if username != requester {
		handleError(c, http.StatusForbidden, "You can only update your own profile", nil)
		return
	}

	var req models.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", username).First(&account).Error; err != nil {
		handleError(c, http.StatusNotFound, "Account not found", err)
		return
	}

	// Update only provided fields
	updates := make(map[string]interface{})
	if req.Bio != "" {
		updates["bio"] = req.Bio
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}
	if len(updates) == 0 {
		handleError(c, http.StatusBadRequest, "No fields to update", nil)
		return
	}

	if err := db.Model(&account).Updates(updates).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to update profile", err)
		return
	}

	c.JSON(http.StatusOK, account)
}

// GetAccountEventHistory returns a user's event/activity history
func GetAccountEventHistory(c *gin.Context) {
	username := c.Param("username")
	db := database.GetDB()

	var activities []models.ActivityLog
	if err := db.Where("username = ?", username).Order("timestamp DESC").Find(&activities).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to fetch event history", err)
		return
	}

	c.JSON(http.StatusOK, activities)
}

// ListNotifications returns recent notifications for the logged-in user
func ListNotifications(c *gin.Context) {
	username := c.GetString("username")
	db := database.GetDB()

	var notifications []models.Notification
	query := db.Where("recipient_username = ?", username).Order("created_at DESC")

	if c.Query("unread") == "true" {
		query = query.Where("read = ?", false)
	}
	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	query = query.Limit(limit)

	if err := query.Find(&notifications).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to fetch notifications", err)
		return
	}

	// Mark unread notifications as read if any are returned
	if len(notifications) > 0 {
		var ids []uint
		for _, n := range notifications {
			if !n.Read {
				ids = append(ids, n.ID)
			}
		}
		if len(ids) > 0 {
			db.Model(&models.Notification{}).Where("id IN ?", ids).Update("read", true)
		}
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadNotificationCount returns the unread notification count for the logged-in user
func GetUnreadNotificationCount(c *gin.Context) {
	username := c.GetString("username")
	db := database.GetDB()

	var count int64
	if err := db.Model(&models.Notification{}).Where("recipient_username = ? AND read = ?", username, false).Count(&count).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to fetch unread count", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}
