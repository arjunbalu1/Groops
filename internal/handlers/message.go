package handlers

import (
	"encoding/json"
	"groops/internal/database"
	"groops/internal/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetGroupMessages handles fetching messages for a group
func GetGroupMessages(c *gin.Context) {
	groupID := c.Param("group_id")
	requester := c.GetString("username")

	if requester == "" {
		log.Printf("Error: Not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	db := database.GetDB()

	// Check if group exists and user is a member
	var group models.Group
	if err := db.Preload("Members").Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if user is organizer or approved member
	isMember := group.OrganiserID == requester
	if !isMember {
		for _, member := range group.Members {
			if member.Username == requester && member.Status == "approved" {
				isMember = true
				break
			}
		}
	}

	if !isMember {
		log.Printf("Error: User %s not authorized to view messages for group %s", requester, groupID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view group messages"})
		return
	}

	// Get pagination parameters
	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Get the before parameter for pagination (message ID to get messages before)
	var beforeID uint = 0
	if beforeStr := c.Query("before"); beforeStr != "" {
		if parsedBefore, err := strconv.ParseUint(beforeStr, 10, 32); err == nil {
			beforeID = uint(parsedBefore)
		}
	}

	// Build query for fetching messages
	query := db.Where("group_id = ?", groupID)

	// If beforeID is provided, get messages with ID less than it (older messages)
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}

	// Fetch messages ordered by creation time (newest first for pagination, frontend will reverse)
	var messages []models.Message
	if err := query.Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		log.Printf("Error: Failed to fetch messages for group %s: %v", groupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	// Mark messages as read by this user
	if len(messages) > 0 {
		for i := range messages {
			// Parse existing ReadBy array
			var readByUsers []string
			if messages[i].ReadBy != nil {
				if err := json.Unmarshal(messages[i].ReadBy, &readByUsers); err != nil {
					log.Printf("Warning: Failed to parse ReadBy for message %d: %v", messages[i].ID, err)
					readByUsers = []string{}
				}
			}

			// Check if user has already read this message
			hasRead := false
			for _, user := range readByUsers {
				if user == requester {
					hasRead = true
					break
				}
			}

			// If not read yet, add user to ReadBy array and update database
			if !hasRead {
				readByUsers = append(readByUsers, requester)
				updatedReadBy, err := json.Marshal(readByUsers)
				if err != nil {
					log.Printf("Warning: Failed to marshal ReadBy for message %d: %v", messages[i].ID, err)
					continue
				}

				// Update the message in database
				if err := db.Model(&messages[i]).Update("read_by", updatedReadBy).Error; err != nil {
					log.Printf("Warning: Failed to update ReadBy for message %d: %v", messages[i].ID, err)
				}

				// Update the local message object for response
				messages[i].ReadBy = updatedReadBy
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// SendGroupMessage handles sending a message to a group
func SendGroupMessage(c *gin.Context) {
	groupID := c.Param("group_id")
	requester := c.GetString("username")

	if requester == "" {
		log.Printf("Error: Not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var request models.SendMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("Error: Invalid message input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message content"})
		return
	}

	// Additional validation
	if len(request.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message cannot be empty"})
		return
	}

	db := database.GetDB()

	// Check if group exists and user is a member
	var group models.Group
	if err := db.Preload("Members").Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if user is organizer or approved member
	isMember := group.OrganiserID == requester
	if !isMember {
		for _, member := range group.Members {
			if member.Username == requester && member.Status == "approved" {
				isMember = true
				break
			}
		}
	}

	if !isMember {
		log.Printf("Error: User %s not authorized to send messages to group %s", requester, groupID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to send messages to this group"})
		return
	}

	// Create the message
	message := models.Message{
		GroupID:  groupID,
		Username: requester,
		Content:  request.Content,
	}

	// Initialize ReadBy with the sender (they've "read" their own message)
	readByUsers := []string{requester}
	readByJSON, err := json.Marshal(readByUsers)
	if err != nil {
		log.Printf("Warning: Failed to marshal initial ReadBy: %v", err)
		readByJSON = []byte("[]") // Fallback to empty array
	}
	message.ReadBy = readByJSON

	if err := db.Create(&message).Error; err != nil {
		log.Printf("Error: Failed to create message for group %s: %v", groupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Log the activity
	if err := LogActivity(requester, "send_message", groupID); err != nil {
		log.Printf("Warning: Failed to log message activity: %v", err)
	}

	// Create unread message notifications after 10 seconds (async)
	go func() {
		time.Sleep(10 * time.Second)

		// Get all group members (organizer + approved members)
		var allMembers []string

		// Add organizer
		allMembers = append(allMembers, group.OrganiserID)

		// Add approved members
		for _, member := range group.Members {
			if member.Status == "approved" && member.Username != group.OrganiserID {
				allMembers = append(allMembers, member.Username)
			}
		}

		// For each member (except the sender), check if they need an unread_messages notification
		for _, memberUsername := range allMembers {
			if memberUsername == requester {
				continue // Skip the sender
			}

			// Check if this member has unread messages in this group
			var unreadCount int64
			query := `
				SELECT COUNT(*) 
				FROM message 
				WHERE group_id = ? 
				AND (read_by IS NULL OR NOT jsonb_exists(read_by, ?))
			`

			if err := db.Raw(query, groupID, memberUsername).Scan(&unreadCount).Error; err != nil {
				log.Printf("Warning: Failed to count unread messages for %s: %v", memberUsername, err)
				continue
			}

			// If they have unread messages, check if they already have an unread_messages notification for this group
			if unreadCount > 0 {
				var existingNotifCount int64
				if err := db.Model(&models.Notification{}).
					Where("recipient_username = ? AND type = ? AND group_id = ? AND read = ?",
						memberUsername, "unread_messages", groupID, false).
					Count(&existingNotifCount).Error; err != nil {
					log.Printf("Warning: Failed to check existing notifications for %s: %v", memberUsername, err)
					continue
				}

				// Only create notification if they don't already have one for this group
				if existingNotifCount == 0 {
					notificationMsg := "You have unread messages in '" + group.Name + "'"
					if err := createNotification(db, memberUsername, "unread_messages", notificationMsg, groupID); err != nil {
						log.Printf("Warning: Failed to create unread messages notification for %s: %v", memberUsername, err)
					}
				}
			}
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": message,
		"success": true,
	})
}
