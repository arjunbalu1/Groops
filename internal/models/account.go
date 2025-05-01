package models

import (
	"time"

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
	GoogleID      string        `gorm:"uniqueIndex;size:128;not null" json:"google_id"`
	Username      string        `gorm:"primaryKey;size:30;not null" json:"username" binding:"required,alphanum"`
	Email         string        `gorm:"uniqueIndex;size:255;not null" json:"email" binding:"required,email"`
	EmailVerified bool          `gorm:"not null;default:false" json:"email_verified"`
	FullName      string        `gorm:"size:255" json:"full_name"`
	GivenName     string        `gorm:"size:100" json:"given_name"`
	FamilyName    string        `gorm:"size:100" json:"family_name"`
	Locale        string        `gorm:"size:10" json:"locale"`
	DateJoined    time.Time     `gorm:"not null" json:"date_joined"`
	Rating        float64       `gorm:"type:decimal(3,2);not null;default:5.0" json:"rating"`
	Bio           string        `gorm:"type:text" json:"bio"`
	AvatarURL     string        `gorm:"size:512" json:"avatar_url"`
	Activities    []ActivityLog `gorm:"foreignKey:Username" json:"activities"`
	OwnedGroups   []Group       `gorm:"foreignKey:OrganiserID" json:"owned_groups"`
	JoinedGroups  []GroupMember `gorm:"foreignKey:Username" json:"joined_groups"`
	LastLogin     time.Time     `gorm:"not null" json:"last_login"`
	CreatedAt     time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time     `gorm:"not null" json:"updated_at"`
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

// CreateAccountRequest represents the data needed to create a new account
type CreateAccountRequest struct {
	Username  string `json:"username" binding:"required,alphanum,min=3,max=30"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
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

// LoginLog represents a user login/logout history record
type LoginLog struct {
	ID         uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Username   string     `gorm:"size:30;not null;index" json:"username"`
	GoogleID   string     `gorm:"size:128;not null" json:"google_id"`
	LoginTime  time.Time  `gorm:"not null;index" json:"login_time"`
	LogoutTime *time.Time `json:"logout_time"` // Nullable - will be null until logout
	IPAddress  string     `gorm:"size:45" json:"ip_address"`
	UserAgent  string     `gorm:"size:255" json:"user_agent"`
	SessionID  string     `gorm:"size:64;uniqueIndex" json:"session_id"`
	IsTemp     bool       `gorm:"not null" json:"is_temp"` // Flag for temp accounts
}
