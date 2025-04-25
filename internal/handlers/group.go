package handlers

import (
	"errors"
	"fmt"
	"groops/internal/database"
	"groops/internal/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateGroup handles the creation of a new group
func CreateGroup(c *gin.Context) {
	var request models.CreateGroupRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		handleError(c, http.StatusBadRequest, fmt.Sprintf("Invalid input: %s", err.Error()), err)
		return
	}

	// Validate that DateTime is in the future
	if request.DateTime.Before(time.Now()) {
		handleError(c, http.StatusBadRequest, "Event date must be in the future",
			fmt.Errorf("event date %v is before current time", request.DateTime))
		return
	}

	db := database.GetDB()

	// Find the organizer account
	var organizer models.Account
	if err := db.Where("username = ?", request.OrganizerUsername).First(&organizer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleError(c, http.StatusNotFound, "Organizer not found", err)
			return
		}
		handleError(c, http.StatusInternalServerError, "Unable to verify organizer", err)
		return
	}

	// Create the group
	group := models.Group{
		Name:         request.Name,
		DateTime:     request.DateTime,
		Venue:        request.Venue,
		Cost:         request.Cost,
		SkillLevel:   models.SkillLevel(request.SkillLevel),
		ActivityType: models.ActivityType(request.ActivityType),
		MaxMembers:   request.MaxMembers,
		Description:  request.Description,
		OrganiserID:  organizer.Username,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&group).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to create group", err)
		return
	}

	// Create the first group member (organizer)
	member := models.GroupMember{
		GroupID:   group.ID,
		Username:  organizer.Username,
		Status:    "approved",
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.Create(&member).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to add organizer as member", err)
		return
	}

	// Log the activity
	if err := LogActivity(organizer.Username, "create_group", group.ID); err != nil {
		// Log but don't fail the request
		log.Printf("Warning: Failed to log activity: %v", err)
	}

	c.JSON(http.StatusCreated, group)
}

// GetGroups handles listing all groups
func GetGroups(c *gin.Context) {
	db := database.GetDB()
	var groups []models.Group

	if err := db.Preload("Members").Find(&groups).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to fetch groups", err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// LogActivity adds a new activity to user's history with retry logic
func LogActivity(username string, eventType string, groupID string) error {
	activity := models.ActivityLog{
		Username:  username,
		EventType: eventType,
		GroupID:   groupID,
		Timestamp: time.Now(),
	}

	db := database.GetDB()

	// Try up to 3 times
	var err error
	for attempts := 0; attempts < 3; attempts++ {
		if err = db.Create(&activity).Error; err != nil {
			log.Printf("Failed to log activity (attempt %d/3): %v", attempts+1, err)
			time.Sleep(time.Second * time.Duration(attempts+1)) // Backoff
			continue
		}
		return nil
	}

	// If we got here, all attempts failed
	return fmt.Errorf("failed to log activity after 3 attempts: %v", err)
}

// JoinGroup handles a user's request to join a group
func JoinGroup(c *gin.Context) {
	groupID := c.Param("group_id")
	username := c.GetString("username") // Set by auth middleware

	db := database.GetDB()

	// Check if group exists
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		handleError(c, http.StatusNotFound, "Group not found", err)
		return
	}

	// Check if user is already a member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ?", groupID, username).First(&member).Error; err == nil {
		if member.Status == "approved" {
			handleError(c, http.StatusConflict, "Already a member", nil)
			return
		} else if member.Status == "pending" {
			handleError(c, http.StatusConflict, "Join request already pending", nil)
			return
		}
	}

	// Check if group is full (approved members)
	var approvedCount int64
	db.Model(&models.GroupMember{}).Where("group_id = ? AND status = ?", groupID, "approved").Count(&approvedCount)
	if int(approvedCount) >= group.MaxMembers {
		handleError(c, http.StatusForbidden, "Group is full", nil)
		return
	}

	// Create join request (pending status)
	newMember := models.GroupMember{
		GroupID:   groupID,
		Username:  username,
		Status:    "pending",
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(&newMember).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to request to join group", err)
		return
	}

	// Log activity
	_ = LogActivity(username, "join_group_request", groupID)

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
		handleError(c, http.StatusNotFound, "Group not found", err)
		return
	}

	// Prevent organizer from leaving their own group
	if group.OrganiserID == username {
		handleError(c, http.StatusForbidden, "Organizer cannot leave their own group", nil)
		return
	}

	// Check if user is a member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ?", groupID, username).First(&member).Error; err != nil {
		handleError(c, http.StatusNotFound, "Not a group member", err)
		return
	}

	// Only allow approved or pending members to leave
	if member.Status != "approved" && member.Status != "pending" {
		handleError(c, http.StatusForbidden, "Cannot leave group with current status", nil)
		return
	}

	// Remove membership (delete row)
	if err := db.Delete(&member).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to leave group", err)
		return
	}

	// Log activity
	_ = LogActivity(username, "leave_group", groupID)

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

// ListPendingMembers returns all pending join requests for a group (organiser only)
func ListPendingMembers(c *gin.Context) {
	groupID := c.Param("group_id")
	requester := c.GetString("username")

	db := database.GetDB()

	// Check if group exists and get organiser
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		handleError(c, http.StatusNotFound, "Group not found", err)
		return
	}

	// Only organiser can view pending members
	if group.OrganiserID != requester {
		handleError(c, http.StatusForbidden, "Only organiser can view pending members", nil)
		return
	}

	var pendingMembers []models.GroupMember
	if err := db.Where("group_id = ? AND status = ?", groupID, "pending").Find(&pendingMembers).Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to fetch pending members", err)
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

	// Check if group exists and get organiser
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		handleError(c, http.StatusNotFound, "Group not found", err)
		return
	}
	if group.OrganiserID != requester {
		handleError(c, http.StatusForbidden, "Only organiser can approve members", nil)
		return
	}

	// Find the pending member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ? AND status = ?", groupID, username, "pending").First(&member).Error; err != nil {
		handleError(c, http.StatusNotFound, "Pending join request not found", err)
		return
	}

	// Check if group is full (approved members)
	var approvedCount int64
	db.Model(&models.GroupMember{}).Where("group_id = ? AND status = ?", groupID, "approved").Count(&approvedCount)
	if int(approvedCount) >= group.MaxMembers {
		handleError(c, http.StatusForbidden, "Group is full", nil)
		return
	}

	// Approve the member
	if err := db.Model(&member).Update("status", "approved").Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to approve member", err)
		return
	}

	_ = LogActivity(username, "join_group_approved", groupID)
	c.JSON(http.StatusOK, gin.H{"message": "Member approved"})
}

// RejectJoinRequest allows organiser to reject a pending join request
func RejectJoinRequest(c *gin.Context) {
	groupID := c.Param("group_id")
	username := c.Param("username")
	requester := c.GetString("username")

	db := database.GetDB()

	// Check if group exists and get organiser
	var group models.Group
	if err := db.Where("id = ?", groupID).First(&group).Error; err != nil {
		handleError(c, http.StatusNotFound, "Group not found", err)
		return
	}
	if group.OrganiserID != requester {
		handleError(c, http.StatusForbidden, "Only organiser can reject members", nil)
		return
	}

	// Find the pending member
	var member models.GroupMember
	if err := db.Where("group_id = ? AND username = ? AND status = ?", groupID, username, "pending").First(&member).Error; err != nil {
		handleError(c, http.StatusNotFound, "Pending join request not found", err)
		return
	}

	// Reject the member (update status or delete row)
	if err := db.Model(&member).Update("status", "rejected").Error; err != nil {
		handleError(c, http.StatusInternalServerError, "Failed to reject member", err)
		return
	}

	_ = LogActivity(username, "join_group_rejected", groupID)
	c.JSON(http.StatusOK, gin.H{"message": "Member rejected"})
}
