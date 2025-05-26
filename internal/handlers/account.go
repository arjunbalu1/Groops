package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"groops/internal/auth"
	"groops/internal/database"
	"groops/internal/models"
	"groops/internal/services"

	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetMyProfile retrieves the current authenticated user's profile
func GetMyProfile(c *gin.Context) {
	username := c.GetString("username")

	// Check if user is authenticated
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":         "Not authenticated",
			"authenticated": false,
		})
		return
	}

	// Check if user needs to complete profile (temp username)
	if len(username) > 5 && username[:5] == "temp-" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":         "Profile incomplete",
			"authenticated": true,
			"needsProfile":  true,
			"username":      username,
			"email":         c.GetString("email"),
			"name":          c.GetString("name"),
			"picture":       c.GetString("picture"),
			"given_name":    c.GetString("given_name"),
			"family_name":   c.GetString("family_name"),
		})
		return
	}

	// Get full account data
	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":         "Account not found",
				"authenticated": false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":         "Failed to retrieve account",
			"authenticated": false,
		})
		return
	}

	// Return user profile data
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"needsProfile":  false,
		"username":      account.Username,
		"email":         account.Email,
		"fullName":      account.FullName,
		"givenName":     account.GivenName,
		"familyName":    account.FamilyName,
		"bio":           account.Bio,
		"avatarURL":     account.AvatarURL,
		"rating":        account.Rating,
		"dateJoined":    account.DateJoined,
		"lastLogin":     account.LastLogin,
		"emailVerified": account.EmailVerified,
		"locale":        account.Locale,
	})
}

// GetAccount retrieves account information (only allows access to own account)
func GetAccount(c *gin.Context) {
	username := c.Param("username")
	requestingUser := c.GetString("username") // Get the authenticated user

	// Authorization check: only allow users to access their own account
	if username != requestingUser {
		log.Printf("Error: User %s attempted to access account %s", requestingUser, username)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - can only access your own account"})
		return
	}

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

	// Check if username is already taken by someone else (case-insensitive)
	var existingUsername models.Account
	if err := db.Where("LOWER(username) = LOWER(?) AND google_id != ?", req.Username, sub).First(&existingUsername).Error; err == nil {
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
		// Store the old username before updating
		oldUsername := tempAccount.Username

		// Update the temporary account with the chosen username and other details
		updates := map[string]interface{}{
			"username":       req.Username,
			"bio":            req.Bio,
			"avatar_url":     avatarURL,
			"updated_at":     now,
			"full_name":      req.FullName,
			"given_name":     givenName,
			"family_name":    familyName,
			"locale":         locale,
			"email_verified": emailVerified,
		}

		// If FullName is not provided, fallback to Google name
		if req.FullName == "" {
			updates["full_name"] = name
		}

		if err := db.Model(&tempAccount).Updates(updates).Error; err != nil {
			log.Printf("Error: Failed to update account: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
			return
		}

		// Get the final chosen name (either provided by user or Google default)
		chosenName := req.FullName
		if chosenName == "" {
			chosenName = name
		}

		// Update all related tables with the new username
		// This ensures we don't lose relationships when username changes

		// 1. Update group memberships
		if err := db.Model(&models.GroupMember{}).Where("username = ?", oldUsername).Update("username", req.Username).Error; err != nil {
			log.Printf("Warning: Failed to update group memberships: %v", err)
			// Non-fatal error - continue
		}

		// 2. Update activity logs
		if err := db.Model(&models.ActivityLog{}).Where("username = ?", oldUsername).Update("username", req.Username).Error; err != nil {
			log.Printf("Warning: Failed to update activity logs: %v", err)
			// Non-fatal error - continue
		}

		// 3. Update notifications
		if err := db.Model(&models.Notification{}).Where("recipient_username = ?", oldUsername).Update("recipient_username", req.Username).Error; err != nil {
			log.Printf("Warning: Failed to update notifications: %v", err)
			// Non-fatal error - continue
		}

		// 4. Update login logs
		if err := db.Model(&models.LoginLog{}).Where("username = ?", oldUsername).Update("username", req.Username).Error; err != nil {
			log.Printf("Warning: Failed to update login logs: %v", err)
			// Non-fatal error - continue
		}

		// 4a. Update the name in the login log if it differs from the Google name
		if chosenName != name {
			if err := db.Model(&models.LoginLog{}).Where("session_id = ?", sessionID).Update("name", chosenName).Error; err != nil {
				log.Printf("Warning: Failed to update login log name: %v", err)
				// Non-fatal error - continue
			}
		}

		// 5. Update session directly in the database
		// Also update the name in the session if it differs from the Google name
		sessionUpdates := map[string]interface{}{
			"username": req.Username,
		}
		if chosenName != name {
			sessionUpdates["name"] = chosenName
		}
		if err := db.Model(&models.Session{}).Where("id = ?", sessionID).Updates(sessionUpdates).Error; err != nil {
			log.Printf("Warning: Failed to update session: %v", err)
			// Non-fatal error - continue
		} else {
			log.Printf("Session %s updated with new username: %s and name: %s", sessionID, req.Username, chosenName)
		}

		// Retrieve the updated account
		if err := db.Where("google_id = ?", sub).First(&tempAccount).Error; err != nil {
			log.Printf("Error: Failed to retrieve updated account: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated account"})
			return
		}

		// Send welcome email to the user
		emailSvc := services.NewEmailService()
		if err := emailSvc.SendWelcomeEmail(email, chosenName); err != nil {
			log.Printf("Warning: Failed to send welcome email: %v", err)
			// Non-fatal error - continue with the response
		} else {
			log.Printf("Welcome email sent to %s (%s)", chosenName, email)
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
	username := c.GetString("username")

	// Check if username exists in the session
	if username == "" {
		log.Printf("Error: No authenticated user found")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
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

// GetPublicProfile retrieves safe, public profile information
// This endpoint is safe to expose publicly as it only returns non-sensitive data
func GetPublicProfile(c *gin.Context) {
	username := c.Param("username")

	db := database.GetDB()
	var account models.Account
	// Use case-insensitive username lookup to prevent duplicate usernames with different cases
	if err := db.Where("LOWER(username) = LOWER(?)", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("Error: Account not found: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			return
		}
		log.Printf("Error: Failed to retrieve account: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		return
	}

	// Return only safe, public information
	publicProfile := gin.H{
		"username":    account.Username,
		"full_name":   account.FullName,
		"avatar_url":  account.AvatarURL,
		"bio":         account.Bio,
		"rating":      account.Rating,
		"date_joined": account.DateJoined,
	}

	c.JSON(http.StatusOK, publicProfile)
}

// GetProfileImage proxies profile images to avoid CORS issues
func GetProfileImage(c *gin.Context) {
	username := c.Param("username")

	db := database.GetDB()
	var account models.Account
	if err := db.Where("username = ?", username).First(&account).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	if account.AvatarURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No avatar available"})
		return
	}

	// Fetch the image from the external URL
	resp, err := http.Get(account.AvatarURL)
	if err != nil {
		log.Printf("Error fetching image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch image"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Set appropriate headers
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Copy the image data to the response
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		log.Printf("Error copying image data: %v", err)
	}
}
