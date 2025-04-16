package handlers

import (
	"errors"
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
		Cost              float64      `json:"cost" binding:"required"`
		SkillLevel        string       `json:"skill_level" binding:"required"`
		ActivityType      string       `json:"activity_type" binding:"required"`
		MaxMembers        int          `json:"max_members" binding:"required"`
		Description       string       `json:"description" binding:"required"`
		OrganizerUsername string       `json:"organizer_username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	// Find the organizer account
	var organizer models.Account
	if err := db.Where("username = ?", request.OrganizerUsername).First(&organizer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "organizer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find organizer"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add organizer as member"})
		return
	}

	// Log the activity
	activity := models.ActivityLog{
		Username:  organizer.Username,
		EventType: "create_group",
		GroupID:   group.ID,
		Timestamp: time.Now(),
	}

	if err := db.Create(&activity).Error; err != nil {
		// Log error but don't fail the request
		log.Printf("failed to log activity: %v", err)
	}

	c.JSON(http.StatusCreated, group)
}

// GetGroups handles listing all groups
func GetGroups(c *gin.Context) {
	// TODO: Fetch from database
	groups := []models.Group{
		{
			ID:       "arjun-20240313150000",
			Name:     "Basketball Game",
			DateTime: time.Now().Add(24 * time.Hour),
			Venue: models.Venue{
				FormattedAddress: "123 Sports Center, Downtown, City",
				PlaceID:          "ChIJxxx...",
				Latitude:         40.7128,
				Longitude:        -74.0060,
			},
			Cost:         15.0,
			SkillLevel:   models.Intermediate,
			ActivityType: models.SportActivity,
			MaxMembers:   12,
			Description:  "Weekly basketball game - all welcome!",
			OrganiserID:  "arjun",
			Members: []models.GroupMember{
				{
					GroupID:   "arjun-20240313150000",
					Username:  "arjun",
					Status:    "approved",
					JoinedAt:  time.Now(),
					UpdatedAt: time.Now(),
				},
			},
		},
	}

	c.JSON(http.StatusOK, groups)
}
