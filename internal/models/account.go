package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ActivityLog represents an entry in the user's activity history
type ActivityLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"size:30;not null;index" json:"username"`
	EventType string    `gorm:"size:20;not null" json:"event_type"` // create_group, join_group, leave_group
	GroupID   string    `gorm:"size:50;not null" json:"group_id"`
	Timestamp time.Time `gorm:"not null;index" json:"timestamp"`
}

// ActivityLogList represents a list of activity logs
type ActivityLogList []ActivityLog

func (a *ActivityLogList) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *ActivityLogList) Scan(value interface{}) error {
	if value == nil {
		*a = make([]ActivityLog, 0)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("unsupported type for ActivityLogList: %T", value)
	}
}

// Account represents a user account in the system
type Account struct {
	Username     string         `gorm:"primaryKey;size:30;not null" json:"username" binding:"required,alphanum"`
	Email        string         `gorm:"uniqueIndex;size:255;not null" json:"email" binding:"required,email"`
	HashedPass   string         `gorm:"size:255;not null" json:"-"`
	DateJoined   time.Time      `gorm:"not null" json:"date_joined"`
	Rating       float64        `gorm:"type:decimal(3,2);not null;default:5.0" json:"rating"`
	Activities   []ActivityLog  `gorm:"foreignKey:Username" json:"activities"`
	OwnedGroups  []Group        `gorm:"foreignKey:OrganiserID" json:"owned_groups"`
	JoinedGroups []GroupMember  `gorm:"foreignKey:Username" json:"joined_groups"`
	LastLogin    time.Time      `gorm:"not null" json:"last_login"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// StringList represents a list of strings that can be stored as JSONB
type StringList []string

func (s *StringList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = make([]string, 0)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type for StringList: %T", value)
	}
}

// BeforeCreate hook is called before creating a new account
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	if a.DateJoined.IsZero() {
		a.DateJoined = now
	}
	if a.LastLogin.IsZero() {
		a.LastLogin = now
	}
	if a.Rating == 0 {
		a.Rating = 5.0
	}
	return nil
}

// BeforeSave hook is called before saving the account
func (a *Account) BeforeSave(tx *gorm.DB) error {
	a.UpdatedAt = time.Now()
	return nil
}

// TableName specifies the table name for the Account model
func (Account) TableName() string {
	return "account"
}

// TableName specifies the table name for the ActivityLog model
func (ActivityLog) TableName() string {
	return "activity_log"
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
