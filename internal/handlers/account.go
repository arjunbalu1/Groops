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
	// Check for existing profile by GoogleID or username
	var existing models.Account
	if err := db.Where("google_id = ? OR username = ?", sub, req.Username).First(&existing).Error; err == nil {
		handleError(c, http.StatusConflict, "Profile already exists for this user or username taken", nil)
		return
	}

	// If no avatar URL is provided, use the Google profile picture
	avatarURL := req.AvatarURL
	if avatarURL == "" {
		avatarURL = picture
	}

	now := time.Now()
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
