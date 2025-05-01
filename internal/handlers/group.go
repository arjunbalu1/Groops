package handlers

import (
	"errors"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateGroup handles the creation of a new group
func CreateGroup(c *gin.Context) {
	var request models.CreateGroupRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Printf("Error: Invalid input: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid input: %s", err.Error())})
		return
	}

	// Validate that DateTime is in the future
	if request.DateTime.Before(time.Now()) {
		log.Printf("Error: Event date %v is before current time", request.DateTime)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event date must be in the future"})
		return
	}

	// Get the authenticated username from context
	organizerUsername := c.GetString("username")
	if organizerUsername == "" {
		log.Printf("Error: Not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	db := database.GetDB()

	// Find the organizer account
	var organizer models.Account
	if err := db.Where("username = ?", organizerUsername).First(&organizer).Error; err != nil {
		log.Printf("Error: Organizer not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Organizer not found"})
		return
	}

	// Create the group (use organizerUsername, not request.OrganizerUsername)
	group := models.Group{
		Name:         request.Name,
		DateTime:     request.DateTime,
		Venue:        request.Venue,
		Cost:         request.Cost,
		SkillLevel:   models.SkillLevel(request.SkillLevel),
		ActivityType: models.ActivityType(request.ActivityType),
		MaxMembers:   request.MaxMembers,
		Description:  request.Description,
		OrganiserID:  organizerUsername,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&group).Error; err != nil {
		log.Printf("Error: Failed to create group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	// Create the first group member (organizer)
	member := models.GroupMember{
		GroupID:   group.ID,
		Username:  organizerUsername,
		Status:    "approved",
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.Create(&member).Error; err != nil {
		log.Printf("Error: Failed to add organizer as member: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add organizer as member"})
		return
	}

	// Log the activity
	if err := LogActivity(organizerUsername, "create_group", group.ID); err != nil {
		log.Printf("Warning: Failed to log activity: %v", err)
	}

	c.JSON(http.StatusCreated, group)
}

// GetGroups handles listing all groups with filtering, sorting, and pagination
func GetGroups(c *gin.Context) {
	db := database.GetDB()
	var groups []models.Group

	query := db.Preload("Members")

	// Filtering
	if activityType := c.Query("activity_type"); activityType != "" {
		query = query.Where("activity_type = ?", activityType)
	}
	if skillLevel := c.Query("skill_level"); skillLevel != "" {
		query = query.Where("skill_level = ?", skillLevel)
	}
	if minPrice := c.Query("min_price"); minPrice != "" {
		query = query.Where("cost >= ?", minPrice)
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		query = query.Where("cost <= ?", maxPrice)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		query = query.Where("date_time >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		query = query.Where("date_time <= ?", dateTo)
	}
	if organiserID := c.Query("organiser_id"); organiserID != "" {
		query = query.Where("organiser_id = ?", organiserID)
	}
	if minMembers := c.Query("min_members"); minMembers != "" {
		query = query.Where("max_members >= ?", minMembers)
	}
	if maxMembers := c.Query("max_members"); maxMembers != "" {
		query = query.Where("max_members <= ?", maxMembers)
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}

	// Sorting
	sortBy := c.DefaultQuery("sort_by", "date_time")
	sortOrder := c.DefaultQuery("sort_order", "asc")
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Pagination with defaults
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, err1 := strconv.Atoi(limitStr)
	if err1 != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // max limit
	}
	offset, err2 := strconv.Atoi(offsetStr)
	if err2 != nil || offset < 0 {
		offset = 0
	}
	query = query.Limit(limit).Offset(offset)

	if err := query.Find(&groups).Error; err != nil {
		log.Printf("Error: Failed to fetch groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
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
	err := db.Create(&activity).Error
	if err != nil {
		log.Printf("Warning: Failed to log activity: %v", err)
		}
	return err
}

// Helper to create a notification
func createNotification(db *gorm.DB, recipient, notifType, message, groupID string) error {
	notif := models.Notification{
		RecipientUsername: recipient,
		Type:              notifType,
		Message:           message,
		GroupID:           groupID,
		CreatedAt:         time.Now(),
		Read:              false,
	}
	return db.Create(&notif).Error
}

// JoinGroup handles a user's request to join a group
func JoinGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	username := c.GetString("username") // Set by auth middleware

	db := database.GetDB()

	// Check if group exists
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if user is already a member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ?", groupID, username).First(&member).Error; err == nil {
		switch member.Status {
		case "approved":
			log.Printf("Error: Already a member")
			c.JSON(http.StatusConflict, gin.H{"error": "Already a member"})
			return
		case "pending":
			log.Printf("Error: Join request already pending")
			c.JSON(http.StatusConflict, gin.H{"error": "Join request already pending"})
			return
		case "rejected":
			// Update status to pending and update timestamps
			member.Status = "pending"
			member.UpdatedAt = time.Now()
			member.JoinedAt = time.Now()
			if err := db.Save(&member).Error; err != nil {
				log.Printf("Error: Failed to re-request to join group: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to re-request to join group"})
				return
			}
			// Log activity, notify organiser, etc.
			if err := LogActivity(username, "join_group_request", groupID); err != nil {
				log.Printf("Warning: Failed to log join request activity: %v", err)
			}
			msg := username + " requested to join your group '" + group.Name + "'"
			if err := createNotification(db, group.OrganiserID, "join_request", msg, groupID); err != nil {
				log.Printf("Warning: Failed to create notification: %v", err)
			}
			c.JSON(http.StatusCreated, gin.H{"message": "Join request re-submitted"})
			return
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error: Failed to check group membership: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check group membership"})
		return
	}

	// Check if group is full (approved members)
	var approvedCount int64
	db.Model(&models.GroupMember{}).Where("group_id = ? AND status = ?", groupID, "approved").Count(&approvedCount)
	if int(approvedCount) >= group.MaxMembers {
		log.Printf("Error: Group is full")
		c.JSON(http.StatusForbidden, gin.H{"error": "Group is full"})
		return
	}

	// If not a member, create join request (pending status)
	newMember := models.GroupMember{
		GroupID:   groupID,
		Username:  username,
		Status:    "pending",
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(&newMember).Error; err != nil {
		log.Printf("Error: Failed to request to join group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to request to join group"})
		return
	}

	// Log activity, notify organiser, etc.
	if err := LogActivity(username, "join_group_request", groupID); err != nil {
		log.Printf("Warning: Failed to log join request activity: %v", err)
	}
	msg := username + " requested to join your group '" + group.Name + "'"
	if err := createNotification(db, group.OrganiserID, "join_request", msg, groupID); err != nil {
		log.Printf("Warning: Failed to create notification: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Join request submitted"})
}

// LeaveGroup handles a user's request to leave a group
func LeaveGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	username := c.GetString("username") // Set by auth middleware

	db := database.GetDB()

	// Check if group exists
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Prevent organizer from leaving their own group
	if group.OrganiserID == username {
		log.Printf("Error: Organizer cannot leave their own group")
		c.JSON(http.StatusForbidden, gin.H{"error": "Organizer cannot leave their own group"})
		return
	}

	// Check if user is a member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ?", groupID, username).First(&member).Error; err != nil {
		log.Printf("Error: Not a group member: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Not a group member"})
		return
	}

	// Only allow approved or pending members to leave
	if member.Status != "approved" && member.Status != "pending" {
		log.Printf("Error: Cannot leave group with current status")
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot leave group with current status"})
		return
	}

	// Remove membership (delete row)
	if err := db.Delete(&member).Error; err != nil {
		log.Printf("Error: Failed to leave group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave group"})
		return
	}

	// Log activity
	if err := LogActivity(username, "leave_group", groupID); err != nil {
		log.Printf("Warning: Failed to log leave group activity: %v", err)
	}

	// Notify organiser
	msg := username + " has left your group '" + group.Name + "'"
	if err := createNotification(db, group.OrganiserID, "leave_group", msg, groupID); err != nil {
		log.Printf("Warning: Failed to create leave notification: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

// ListPendingMembers returns all pending join requests for a group (organiser only)
func ListPendingMembers(c *gin.Context) {
	groupID := c.Param("group_id")
	requester := c.GetString("username")

	db := database.GetDB()
	var group models.Group

	// Check if group exists
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if requester is the organizer
	if group.OrganiserID != requester {
		log.Printf("Error: Only the organizer can view pending members")
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organizer can view pending members"})
		return
	}

	var pendingMembers []models.GroupMember
	if err := db.Where("group_id = ? AND status = ?", groupID, "pending").Find(&pendingMembers).Error; err != nil {
		log.Printf("Error: Failed to fetch pending members: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending members"})
		return
	}

	c.JSON(http.StatusOK, pendingMembers)
}

// ApproveJoinRequest allows organiser to approve a pending join request
func ApproveJoinRequest(c *gin.Context) {
	groupID := c.Param("group_id")
	username := c.Param("username")
	requester := c.GetString("username")

	db := database.GetDB()
	var group models.Group

	// Check if group exists
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if requester is the organizer
	if group.OrganiserID != requester {
		log.Printf("Error: Only the organizer can approve members")
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organizer can approve members"})
		return
	}

	// Find the pending member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ? AND status = ?",
		groupID, username, "pending").First(&member).Error; err != nil {
		log.Printf("Error: Pending join request not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending join request not found"})
		return
	}

	// Check if group is full (approved members)
	var approvedCount int64
	db.Model(&models.GroupMember{}).Where("group_id = ? AND status = ?", groupID, "approved").Count(&approvedCount)
	if int(approvedCount) >= group.MaxMembers {
		log.Printf("Error: Group is full")
		c.JSON(http.StatusForbidden, gin.H{"error": "Group is full"})
		return
	}

	// Approve the member
	if err := db.Model(&member).Update("status", "approved").Error; err != nil {
		log.Printf("Error: Failed to approve member: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve member"})
		return
	}

	if err := LogActivity(username, "join_group_approved", groupID); err != nil {
		log.Printf("Warning: Failed to log approve join activity: %v", err)
	}

	// Notify user
	msg := "Your request to join group '" + group.Name + "' was approved"
	if err := createNotification(db, username, "join_approved", msg, groupID); err != nil {
		log.Printf("Warning: Failed to create approval notification: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member approved"})
}

// RejectJoinRequest allows organiser to reject a pending join request
func RejectJoinRequest(c *gin.Context) {
	groupID := c.Param("group_id")
	username := c.Param("username")
	requester := c.GetString("username")

	db := database.GetDB()
	var group models.Group

	// Check if group exists
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if requester is the organizer
	if group.OrganiserID != requester {
		log.Printf("Error: Only the organizer can reject members")
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organizer can reject members"})
		return
	}

	// Find the pending member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ? AND status = ?",
		groupID, username, "pending").First(&member).Error; err != nil {
		log.Printf("Error: Pending join request not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending join request not found"})
		return
	}

	// Reject the member
	if err := db.Model(&member).Update("status", "rejected").Error; err != nil {
		log.Printf("Error: Failed to reject member: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject member"})
		return
	}

	if err := LogActivity(username, "join_group_rejected", groupID); err != nil {
		log.Printf("Warning: Failed to log reject join activity: %v", err)
	}

	// Notify user
	msg := "Your request to join group '" + group.Name + "' was rejected"
	if err := createNotification(db, username, "join_rejected", msg, groupID); err != nil {
		log.Printf("Warning: Failed to create rejection notification: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member rejected"})
}

// GetGroupByID handles fetching a single group's details by ID
func GetGroupByID(c *gin.Context) {
	groupID := c.Param("group_id")
	db := database.GetDB()

	var group models.Group
	// Preload organiser and members
	if err := db.Preload("Members").Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Fetch organiser info
	var organiser models.Account
	if err := db.Where("username = ?", group.OrganiserID).First(&organiser).Error; err != nil {
		log.Printf("Error: Failed to fetch organiser info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organiser info"})
		return
	}

	// Prepare response
	response := gin.H{
		"group": group,
		"organiser": gin.H{
			"username":   organiser.Username,
			"rating":     organiser.Rating,
			"avatar_url": organiser.AvatarURL,
			"bio":        organiser.Bio,
		},
	}

	c.JSON(http.StatusOK, response)
}
