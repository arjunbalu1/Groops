package handlers

import (
	"errors"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"groops/internal/services"
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
		Location:     request.Location,
		Cost:         request.Cost,
		SkillLevel:   request.SkillLevel,
		ActivityType: request.ActivityType,
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

// UpdateGroup handles updating an existing group (organizer only)
func UpdateGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	requester := c.GetString("username")

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

	db := database.GetDB()

	// Check if group exists
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if requester is the organizer
	if group.OrganiserID != requester {
		log.Printf("Error: Only the organizer can update the group")
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organizer can update the group"})
		return
	}

	// Prevent updates if event has already passed
	if time.Now().After(group.DateTime) {
		log.Printf("Error: Attempted to update group after event has ended")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot update group after the event has ended"})
		return
	}

	// Prevent updates if event is happening soon (within 1 hour)
	if time.Until(group.DateTime) < time.Hour {
		log.Printf("Error: Attempted to update group too close to event time")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot update group within 1 hour of the event"})
		return
	}

	// Update the group fields
	group.Name = request.Name
	group.DateTime = request.DateTime
	group.Location = request.Location
	group.Cost = request.Cost
	group.SkillLevel = request.SkillLevel
	group.ActivityType = request.ActivityType
	group.MaxMembers = request.MaxMembers
	group.Description = request.Description

	if err := db.Save(&group).Error; err != nil {
		log.Printf("Error: Failed to update group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
		return
	}

	// Log the activity
	if err := LogActivity(requester, "update_group", groupID); err != nil {
		log.Printf("Warning: Failed to log activity: %v", err)
	}

	c.JSON(http.StatusOK, group)
}

// DeleteGroup handles deleting a group (organizer only)
func DeleteGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	requester := c.GetString("username")

	db := database.GetDB()

	// Check if group exists
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if requester is the organizer
	if group.OrganiserID != requester {
		log.Printf("Error: Only the organizer can delete the group")
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organizer can delete the group"})
		return
	}

	// Prevent deletion if event has already passed
	if time.Now().After(group.DateTime) {
		log.Printf("Error: Attempted to delete group after event has ended")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete group after the event has ended"})
		return
	}

	// Prevent deletion if event is happening soon (within 1 hour)
	if time.Until(group.DateTime) < time.Hour {
		log.Printf("Error: Attempted to delete group too close to event time")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete group within 1 hour of the event"})
		return
	}

	// Start a transaction to delete group and related data
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete all group members
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupMember{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error: Failed to delete group members: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group members"})
		return
	}

	// Delete group notifications
	if err := tx.Where("group_id = ?", groupID).Delete(&models.Notification{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error: Failed to delete notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notifications"})
		return
	}

	// Delete activity logs
	if err := tx.Where("group_id = ?", groupID).Delete(&models.ActivityLog{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error: Failed to delete activity logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete activity logs"})
		return
	}

	// Finally delete the group
	if err := tx.Delete(&group).Error; err != nil {
		tx.Rollback()
		log.Printf("Error: Failed to delete group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("Error: Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	// Log the activity (using the main db since tx is committed)
	if err := LogActivity(requester, "delete_group", groupID); err != nil {
		log.Printf("Warning: Failed to log activity: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}

// GetGroups handles listing all groups with filtering, sorting, and pagination.
// Don't know what's going on here with sort parameter validation and numeric type conversion, but it's SQL injection safe.
func GetGroups(c *gin.Context) {
	db := database.GetDB()
	var groups []models.Group

	query := db.Preload("Members")

	// Location-based distance sorting and filtering
	var userLat, userLng string
	var hasUserLocation bool
	if userLat = c.Query("user_lat"); userLat != "" {
		if userLng = c.Query("user_lng"); userLng != "" {
			hasUserLocation = true
			// Add distance calculation using PostgreSQL's earth distance formula
			// This calculates distance in kilometers using the haversine formula
			query = query.Select(`"group".*, 
				ROUND(
					6371 * acos(
						cos(radians(?)) * 
						cos(radians(CAST(location->>'latitude' AS FLOAT))) * 
						cos(radians(CAST(location->>'longitude' AS FLOAT)) - radians(?)) + 
						sin(radians(?)) * 
						sin(radians(CAST(location->>'latitude' AS FLOAT)))
					)::numeric, 2
				) AS distance_km`, userLat, userLng, userLat)

			// Filter to only show groups within 50km radius using a subquery
			query = query.Where(`(
				6371 * acos(
					cos(radians(?)) * 
					cos(radians(CAST(location->>'latitude' AS FLOAT))) * 
					cos(radians(CAST(location->>'longitude' AS FLOAT)) - radians(?)) + 
					sin(radians(?)) * 
					sin(radians(CAST(location->>'latitude' AS FLOAT)))
				)
			) <= 50`, userLat, userLng, userLat)
		}
	}

	// Search functionality - searches across name, description, activity_type, and organiser_id
	if searchTerm := c.Query("search"); searchTerm != "" {
		searchPattern := "%" + searchTerm + "%"
		query = query.Where(
			"LOWER(name) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?) OR LOWER(activity_type) LIKE LOWER(?) OR LOWER(organiser_id) LIKE LOWER(?)",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Filtering
	if activityType := c.Query("activity_type"); activityType != "" {
		query = query.Where("activity_type = ?", activityType)
	}
	if skillLevel := c.Query("skill_level"); skillLevel != "" {
		query = query.Where("skill_level = ?", skillLevel)
	}
	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			query = query.Where("cost >= ?", price)
		}
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			query = query.Where("cost <= ?", price)
		}
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		query = query.Where("date_time >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		query = query.Where("date_time <= ?", dateTo)
	}
	if minMembers := c.Query("min_members"); minMembers != "" {
		if members, err := strconv.Atoi(minMembers); err == nil {
			query = query.Where("max_members >= ?", members)
		}
	}
	if maxMembers := c.Query("max_members"); maxMembers != "" {
		if members, err := strconv.Atoi(maxMembers); err == nil {
			query = query.Where("max_members <= ?", members)
		}
	}

	// Sorting with validation
	sortBy := c.DefaultQuery("sort_by", "date_time")
	// Validate sort column against allowed values
	validSortColumns := map[string]bool{
		"date_time": true, "name": true, "cost": true,
		"skill_level": true, "activity_type": true, "max_members": true,
		"created_at": true, "updated_at": true,
		"distance": true, // Add distance as valid sort option
	}
	if !validSortColumns[sortBy] {
		sortBy = "date_time" // Default to safe value if invalid
	}

	// Special handling for distance sorting
	if sortBy == "distance" && hasUserLocation {
		// Use the calculated distance_km field for sorting
		sortOrder := c.DefaultQuery("sort_order", "asc")
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "asc"
		}
		query = query.Order(fmt.Sprintf("distance_km %s", sortOrder))
	} else if sortBy == "distance" {
		// If distance sort requested but no user location provided, fallback to date_time
		log.Printf("Warning: Distance sort requested but no user location provided")
		sortBy = "date_time"
		query = query.Order("date_time asc")
	} else {
		// If user location is provided but not sorting by distance, still sort by distance first
		if hasUserLocation {
			sortOrder := c.DefaultQuery("sort_order", "asc")
			if sortOrder != "asc" && sortOrder != "desc" {
				sortOrder = "asc"
			}
			// Always sort by distance first when user location is available, then by requested sort
			query = query.Order(fmt.Sprintf("distance_km asc, %s %s", sortBy, sortOrder))
		} else {
			// Validate sort order
			sortOrder := c.DefaultQuery("sort_order", "asc")
			if sortOrder != "asc" && sortOrder != "desc" {
				sortOrder = "asc" // Default to ascending if invalid
			}
			query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
		}
	}

	// Pagination with defaults
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // max limit
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
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

	// Prevent joining if event has already passed
	if time.Now().After(group.DateTime) {
		log.Printf("Error: Attempted to join group after event has ended")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot join group after the event has ended"})
		return
	}

	// Prevent joining if event is happening soon (within 1 hour)
	if time.Until(group.DateTime) < time.Hour {
		log.Printf("Error: Attempted to join group too close to event time")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot join group within 1 hour of the event"})
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

	// Send email notification to the group organizer
	emailService := services.NewEmailService()
	var organiserAccount models.Account
	if err := db.Where("username = ?", group.OrganiserID).First(&organiserAccount).Error; err != nil {
		log.Printf("Warning: Failed to find organizer account for email: %v", err)
	} else {
		if err := emailService.SendJoinRequestEmail(organiserAccount.Email, group.OrganiserID, username, group.Name); err != nil {
			log.Printf("Warning: Failed to send join request email: %v", err)
		}
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

	// Prevent leaving if event has already passed
	if time.Now().After(group.DateTime) {
		log.Printf("Error: Attempted to leave group after event has ended")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot leave group after the event has ended"})
		return
	}

	// Prevent leaving if event is happening soon (within 1 hour)
	if time.Until(group.DateTime) < time.Hour {
		log.Printf("Error: Attempted to leave group too close to event time")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot leave group within 1 hour of the event"})
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

	// Notify all existing approved group members (except organizer) about the new member
	var existingMembers []models.GroupMember
	if err := db.Where("group_id = ? AND status = ? AND username != ?", groupID, "approved", username).Find(&existingMembers).Error; err != nil {
		log.Printf("Warning: Failed to fetch existing members for new member notifications: %v", err)
	} else {
		memberJoinMsg := username + " has joined your group '" + group.Name + "'"
		for _, existingMember := range existingMembers {
			// Don't notify the organizer (they initiated the approval)
			if existingMember.Username != group.OrganiserID {
				if err := createNotification(db, existingMember.Username, "member_joined", memberJoinMsg, groupID); err != nil {
					log.Printf("Warning: Failed to create member join notification for %s: %v", existingMember.Username, err)
				}
			}
		}
	}

	// Send email notification to the approved user
	emailService := services.NewEmailService()
	var userAccount models.Account
	if err := db.Where("username = ?", username).First(&userAccount).Error; err != nil {
		log.Printf("Warning: Failed to find user account for email: %v", err)
	} else {
		if err := emailService.SendJoinApprovalEmail(userAccount.Email, username, group.Name); err != nil {
			log.Printf("Warning: Failed to send join approval email: %v", err)
		}
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

	// Create frontend-friendly response
	response := gin.H{
		"id":                 group.ID,
		"name":               group.Name,
		"date_time":          group.DateTime,
		"location":           group.Location,
		"cost":               group.Cost,
		"skill_level":        group.SkillLevel,
		"activity_type":      group.ActivityType,
		"max_members":        group.MaxMembers,
		"description":        group.Description,
		"organizer_username": group.OrganiserID,
		"members":            group.Members,
		"created_at":         group.CreatedAt,
		"updated_at":         group.UpdatedAt,
		"organizer": gin.H{
			"username":   organiser.Username,
			"rating":     organiser.Rating,
			"avatar_url": organiser.AvatarURL,
			"bio":        organiser.Bio,
		},
	}

	c.JSON(http.StatusOK, response)
}

// RemoveMember handles the removal of a member from a group by the organizer
func RemoveMember(c *gin.Context) {
	groupID := c.Param("group_id")
	memberUsername := c.Param("username")
	organizerUsername := c.GetString("username")

	db := database.GetDB()

	// Check if group exists
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		log.Printf("Error: Group not found: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Check if requester is the organizer
	if group.OrganiserID != organizerUsername {
		log.Printf("Error: User %s attempted to remove member from group %s but is not the organizer", organizerUsername, groupID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organizer can remove members"})
		return
	}

	// Prevent removal if event has already passed
	if time.Now().After(group.DateTime) {
		log.Printf("Error: Attempted to remove member after event has ended")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove members after the event has ended"})
		return
	}

	// Prevent removal if event is happening soon (within 1 hour)
	if time.Until(group.DateTime) < time.Hour {
		log.Printf("Error: Attempted to remove member too close to event time")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove members within 1 hour of the event"})
		return
	}

	// Check if member exists and is approved
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ? AND status = ?",
		groupID, memberUsername, "approved").First(&member).Error; err != nil {
		log.Printf("Error: Member not found or not approved: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Member not found or not approved"})
		return
	}

	// Don't allow removing the organizer (themselves)
	if memberUsername == organizerUsername {
		log.Printf("Error: Organizer attempted to remove themselves")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organizer cannot be removed"})
		return
	}

	// Delete the member record
	if err := db.Delete(&member).Error; err != nil {
		log.Printf("Error: Failed to remove member: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	// Create notification for the removed member
	notification := models.Notification{
		RecipientUsername: memberUsername,
		Type:              "removed_from_group",
		Message:           fmt.Sprintf("You have been removed from group '%s'", group.Name),
		GroupID:           groupID,
		CreatedAt:         time.Now(),
		Read:              false,
	}

	if err := db.Create(&notification).Error; err != nil {
		log.Printf("Warning: Failed to create notification: %v", err)
	}

	// Get member's email for notification
	var account models.Account
	if err := db.Where("username = ?", memberUsername).First(&account).Error; err == nil {
		emailService := services.NewEmailService()
		go func() {
			if err := emailService.SendMemberRemovalEmail(account.Email, account.Username, group.Name); err != nil {
				log.Printf("Warning: Failed to send email to removed member: %v", err)
			}
		}()
	}

	// Log the activity
	if err := LogActivity(organizerUsername, "remove_member", groupID); err != nil {
		log.Printf("Warning: Failed to log activity: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}
