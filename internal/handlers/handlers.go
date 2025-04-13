package handlers

import (
	"net/http"
	"time"

	"groops/internal/database"
	"groops/internal/models"

	"github.com/gin-gonic/gin"
)

// HomeHandler handles requests to the root path "/"
func HomeHandler(c *gin.Context) {
	c.String(http.StatusOK, "Welcome to Groops!")
}

// HealthHandler is a simple health check endpoint
func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// TestDatabaseHandler tests database operations
func TestDatabaseHandler(c *gin.Context) {
	db := database.GetDB()
	result := make(map[string]interface{})

	// Test 1: Create test account
	testAccount := models.Account{
		Username:   "testuser",
		Email:      "test@example.com",
		HashedPass: "hashed_test_password",
		DateJoined: time.Now(),
		Rating:     5.0,
		ActivityHistory: []models.ActivityLog{
			{
				EventType: "create_group",
				GroopID:   "test-group-1",
				Timestamp: time.Now(),
			},
		},
		OwnedGroops:    []string{"test-group-1"},
		ApprovedGroops: []string{},
		PendingGroops:  []string{},
	}

	if err := db.Create(&testAccount).Error; err != nil {
		result["create_account_error"] = err.Error()
	} else {
		result["create_account"] = "success"
	}

	// Test 2: Create test group
	testGroup := models.Group{
		ID:       "test-group-1",
		Name:     "Test Football Group",
		DateTime: time.Now().Add(24 * time.Hour),
		Venue: models.Venue{
			FormattedAddress: "123 Test Street",
			PlaceID:          "test_place_id",
			Latitude:         40.7128,
			Longitude:        -74.0060,
		},
		Cost:         15.0,
		SkillLevel:   models.Intermediate,
		ActivityType: models.SportActivity,
		MaxMembers:   10,
		Description:  "Test group for database verification",
		OrganiserID:  "testuser",
		Members: []models.Member{
			{
				Username: "testuser",
				Status:   "approved",
			},
		},
	}

	if err := db.Create(&testGroup).Error; err != nil {
		result["create_group_error"] = err.Error()
	} else {
		result["create_group"] = "success"
	}

	// Test 3: Read account back
	var readAccount models.Account
	if err := db.Where("username = ?", "testuser").First(&readAccount).Error; err != nil {
		result["read_account_error"] = err.Error()
	} else {
		result["read_account"] = "success"
		result["account_data"] = readAccount
	}

	// Test 4: Read group back
	var readGroup models.Group
	if err := db.Where("id = ?", "test-group-1").First(&readGroup).Error; err != nil {
		result["read_group_error"] = err.Error()
	} else {
		result["read_group"] = "success"
		result["group_data"] = readGroup
	}

	c.JSON(http.StatusOK, result)
}
