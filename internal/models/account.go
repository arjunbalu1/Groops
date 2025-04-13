package models

import "time"

// Account represents a user account in the system
type Account struct {
	Username        string        `json:"username" binding:"required,alphanum"`
	Email           string        `json:"email" binding:"required,email"`
	HashedPass      string        `json:"-"` // "-" means this field won't be included in JSON
	DateJoined      time.Time     `json:"date_joined"`
	Rating          float64       `json:"rating" binding:"min=1,max=5"`
	ActivityHistory []ActivityLog `json:"activity_history"`
	OwnedGroops     []string      `json:"owned_groops"`    // List of groopIDs owned
	ApprovedGroops  []string      `json:"approved_groops"` // List of groopIDs where membership is approved
	PendingGroops   []string      `json:"pending_groops"`  // List of groopIDs where membership is pending
}

// ActivityLog represents an entry in the user's activity history
type ActivityLog struct {
	EventType string    `json:"event_type"` // "create_group", "join_group", "leave_group"
	GroopID   string    `json:"groop_id"`
	Timestamp time.Time `json:"timestamp"`
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
