package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Account represents a user account in the system
type Account struct {
	Username        string        `json:"username" binding:"required,alphanum"`
	Email           string        `json:"email" binding:"required,email"`
	HashedPass      string        `json:"-"` // "-" means this field won't be included in JSON
	DateJoined      time.Time     `json:"date_joined"`
	Rating          float64       `json:"rating" binding:"min=1,max=5"`
	ActivityHistory ActivityLogList `json:"activity_history" gorm:"type:json"`
	OwnedGroops     StringList      `json:"owned_groops"`    // List of groopIDs owned
	ApprovedGroops  StringList      `json:"approved_groops"` // List of groopIDs where membership is approved
	PendingGroops   StringList      `json:"pending_groops"`  // List of groopIDs where membership is pending
}

type StringList []string

func (s StringList) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal StringList: %v", value)
	}
	return json.Unmarshal(bytes, s)
}

// ActivityLog represents an entry in the user's activity history
type ActivityLog struct {
	EventType string    `json:"event_type"` // "create_group", "join_group", "leave_group"
	GroopID   string    `json:"groop_id"`
	Timestamp time.Time `json:"timestamp"`
}
type ActivityLogList []ActivityLog

func (a ActivityLogList) Value() (driver.Value, error) {
    return json.Marshal(a)
}

func (a *ActivityLogList) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("failed to unmarshal ActivityLogList: %v", value)
    }
    return json.Unmarshal(bytes, a)
}

// CreateAccountRequest represents the data needed to create a new account
type CreateAccountRequest struct {
	Username string `json:"username" binding:"required,alphanum,min=3,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest represents the data needed for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
