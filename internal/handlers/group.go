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
	var request struct {
		Name              string       `json:"name" binding:"required"`
		DateTime          time.Time    `json:"date_time" binding:"required"`
		Venue             models.Venue `json:"venue" binding:"required"`
		Cost              float64      `json:"cost" binding:"required,min=0,max=10000"`
		SkillLevel        string       `json:"skill_level" binding:"required,oneof=beginner intermediate advanced"`
		ActivityType      string       `json:"activity_type" binding:"required,oneof=sport social games other"`
		MaxMembers        int          `json:"max_members" binding:"required,min=2"`
		Description       string       `json:"description" binding:"required,max=1000"`
		OrganizerUsername string       `json:"organizer_username" binding:"required"`
	}

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
