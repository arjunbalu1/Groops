package handlers

import (
	"net/http"
	"time"

	"groops/internal/database"
	"groops/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAccount handles new user registration
func CreateAccount(c *gin.Context) {
	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Hash password properly in production
	now := time.Now()
	account := models.Account{
		Username:   req.Username,
		Email:      req.Email,
		HashedPass: req.Password, // TODO: Implement proper hashing
		DateJoined: now,
		Rating:     5.0,
		LastLogin:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	db := database.GetDB()
	if err := db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
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
	return db.Create(&activity).Error
}

// GetAccount retrieves account information
func GetAccount(c *gin.Context) {
	username := c.Param("username")

	db := database.GetDB()
	var account models.Account
	if err := db.Preload("Activities").Preload("OwnedGroups").Preload("JoinedGroups").
		Where("username = ?", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}
