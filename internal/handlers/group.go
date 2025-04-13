package handlers

import (
	"fmt"
	"groops/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateGroup handles the creation of a new group
func CreateGroup(c *gin.Context) {
	var req models.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate future date (comparing just the dates, ignoring time zones)
	now := time.Now().UTC()
	eventTime := req.DateTime.UTC()
	if eventTime.Before(now) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group date must be in the future"})
		return
	}

	// TODO: Get actual username from authentication
	username := "arjun" // This will come from authenticated session

	// Create new group with organizer as first member
	group := models.Group{
		ID:           fmt.Sprintf("%s-%s", username, time.Now().Format("20060102150405")),
		Name:         req.Name,
		DateTime:     req.DateTime,
		Venue:        req.Venue,
		Cost:         req.Cost,
		SkillLevel:   req.SkillLevel,
		ActivityType: req.ActivityType,
		MaxMembers:   req.MaxMembers,
		Description:  req.Description,
		OrganiserID:  username,
		Members: []models.Member{
			{
				Username: username,
				Status:   "approved", // Organizer is automatically approved
			},
		},
	}

	// TODO: Save to database
	// TODO: Add to organizer's activity history
	// TODO: Add to organizer's owned groups

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
			Members: []models.Member{
				{
					Username: "arjun",
					Status:   "approved",
				},
			},
		},
	}

	c.JSON(http.StatusOK, groups)
}
