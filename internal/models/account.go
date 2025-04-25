package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ActivityLog represents an entry in the user's activity history
type ActivityLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"size:30;not null;index" json:"username"`
	EventType string    `gorm:"size:20;not null" json:"event_type"`
	GroupID   string    `gorm:"size:50;not null" json:"group_id"`
	Timestamp time.Time `gorm:"not null;index" json:"timestamp"`
}

// Account represents a user account in the system
type Account struct {
	Username     string         `gorm:"primaryKey;size:30;not null" json:"username" binding:"required,alphanum"`
	Email        string         `gorm:"uniqueIndex;size:255;not null" json:"email" binding:"required,email"`
	HashedPass   string         `gorm:"size:255;not null" json:"-"`
	DateJoined   time.Time      `gorm:"not null" json:"date_joined"`
	Rating       float64        `gorm:"type:decimal(3,2);not null;default:5.0" json:"rating"`
	Bio          string         `gorm:"type:text" json:"bio"`
	AvatarURL    string         `gorm:"size:512" json:"avatar_url"`
	Activities   []ActivityLog  `gorm:"foreignKey:Username" json:"activities"`
	OwnedGroups  []Group        `gorm:"foreignKey:OrganiserID" json:"owned_groups"`
	JoinedGroups []GroupMember  `gorm:"foreignKey:Username" json:"joined_groups"`
	LastLogin    time.Time      `gorm:"not null" json:"last_login"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
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

	// Hash password if it's not already hashed
	if !isPasswordHashed(a.HashedPass) {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(a.HashedPass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		a.HashedPass = string(hashedPassword)
	}

	return nil
}

// Helper to check if password is already hashed
func isPasswordHashed(password string) bool {
	// Bcrypt hashes start with $2a$, $2b$ or $2y$
	return len(password) > 4 && (password[:4] == "$2a$" ||
		password[:4] == "$2b$" ||
		password[:4] == "$2y$")
}

// VerifyPassword checks if the provided password matches the stored hash
func (a *Account) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.HashedPass), []byte(password))
	return err == nil
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
	Username  string `json:"username" binding:"required,alphanum,min=3,max=30"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
}

// LoginRequest represents the data needed for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateAccountRequest for profile updates
// Only bio and avatar_url are updatable for now
// You can expand this as needed
type UpdateAccountRequest struct {
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
}

// Notification represents a user notification in the system
// Used for in-app notifications (e.g., join requests, approvals, etc.)
type Notification struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	RecipientUsername string    `gorm:"size:30;not null;index" json:"recipient_username"`
	Type              string    `gorm:"size:30;not null" json:"type"`
	Message           string    `gorm:"type:text;not null" json:"message"`
	GroupID           string    `gorm:"size:50" json:"group_id"`
	CreatedAt         time.Time `gorm:"not null" json:"created_at"`
	Read              bool      `gorm:"not null;default:false" json:"read"`
}
