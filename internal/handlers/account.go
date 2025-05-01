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
			log.Printf("Error: Account not found: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			return
		}
		log.Printf("Error: Failed to retrieve account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// CreateProfile handles new Google OAuth user profile registration
func CreateProfile(c *gin.Context) {
	sub := c.GetString("sub")
	email := c.GetString("email")
	picture := c.GetString("picture")
	name := c.GetString("name")
	givenName := c.GetString("given_name")
	familyName := c.GetString("family_name")
	locale := c.GetString("locale")
	emailVerified := c.GetBool("email_verified")

	if sub == "" {
		log.Printf("Error: Missing Google ID in token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing Google ID in token"})
		return
	}

	if email == "" {
		log.Printf("Error: Missing email in token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing email in token"})
		return
	}

	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error: Invalid input: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Get the session
	sessionID, err := c.Cookie(auth.SessionCookieName)
	if err != nil {
		log.Printf("Error: No active session: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	db := database.GetDB()

	// Check if username is already taken by someone else
	var existingUsername models.Account
	if err := db.Where("username = ? AND google_id != ?", req.Username, sub).First(&existingUsername).Error; err == nil {
		log.Printf("Error: Username already taken")
		c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
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
			"username":       req.Username,
			"bio":            req.Bio,
			"avatar_url":     avatarURL,
			"updated_at":     now,
			"full_name":      name,
			"given_name":     givenName,
			"family_name":    familyName,
			"locale":         locale,
			"email_verified": emailVerified,
		}

		if err := db.Model(&tempAccount).Updates(updates).Error; err != nil {
			log.Printf("Error: Failed to update account: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
			return
		}

		// Retrieve the updated account
		if err := db.Where("google_id = ?", sub).First(&tempAccount).Error; err != nil {
			log.Printf("Error: Failed to retrieve updated account: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated account"})
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
		log.Printf("Error: Profile already exists for this user")
		c.JSON(http.StatusConflict, gin.H{"error": "Profile already exists for this user"})
		return
	}

	// No existing account found - this should never happen in normal flow
	log.Printf("Error: No temporary account found for Google ID: %s", sub)
	c.JSON(http.StatusBadRequest, gin.H{"error": "No temporary account found. Please try logging in again."})
}

// UpdateAccount allows a user to update their profile (bio, avatar_url)
func UpdateAccount(c *gin.Context) {
	username := c.Param("username")
	requester := c.GetString("username")

	// Only the user themselves can update their profile
	if username != requester {
		log.Printf("Error: You can only update your own profile")
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own profile"})
		return
	}

	var req models.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error: Invalid input: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", username).First(&account).Error; err != nil {
		log.Printf("Error: Account not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
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
		log.Printf("Error: No fields to update")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := db.Model(&account).Updates(updates).Error; err != nil {
		log.Printf("Error: Failed to update profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
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
		log.Printf("Error: Failed to fetch event history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event history"})
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
		log.Printf("Error: Failed to fetch notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
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
		log.Printf("Error: Failed to fetch unread count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}
