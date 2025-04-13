package handlers

import (
	"net/http"
	"time"

	"groops/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateAccount handles new user registration
func CreateAccount(c *gin.Context) {
	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Check if username already exists
	// TODO: Hash password

	account := models.Account{
		Username:        req.Username,
		Email:           req.Email,
		HashedPass:      "hashed_password_here", // TODO: Implement proper hashing
		DateJoined:      time.Now(),
		Rating:          5.0, // Default rating
		ActivityHistory: []models.ActivityLog{},
		OwnedGroops:     []string{},
		ApprovedGroops:  []string{},
		PendingGroops:   []string{},
	}

	// TODO: Save to database

	c.JSON(http.StatusCreated, account)
}

// LogActivity adds a new activity to user's history
func LogActivity(username string, eventType string, groopID string) {
	// TODO: Implement database operation
	activity := models.ActivityLog{
		EventType: eventType,
		GroopID:   groopID,
		Timestamp: time.Now(),
	}
	_ = activity // Remove when implementing database operations
}

// GetAccount retrieves account information
func GetAccount(c *gin.Context) {
	username := c.Param("username")

	// TODO: Fetch from database
	account := models.Account{
		Username:   username,
		Email:      username + "@example.com",
		DateJoined: time.Now().Add(-24 * time.Hour), // Example: joined yesterday
		Rating:     4.5,
		ActivityHistory: []models.ActivityLog{
			{
				EventType: "create_group",
				GroopID:   "sample-group-1",
				Timestamp: time.Now().Add(-1 * time.Hour),
			},
		},
		OwnedGroops:    []string{"sample-group-1"},
		ApprovedGroops: []string{"sample-group-2"},
		PendingGroops:  []string{"sample-group-3"},
	}

	c.JSON(http.StatusOK, account)
}
