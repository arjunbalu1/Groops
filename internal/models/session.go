package models

import (
	"time"

	"gorm.io/gorm"
)

// SessionDuration is the length of time a session remains valid
const SessionDuration = time.Hour * 24 * 7 // 1 week

// Session represents a user session
type Session struct {
	ID            string    `gorm:"primaryKey;size:64" json:"-"`
	UserID        string    `gorm:"size:128;index" json:"-"`         // Google ID
	Username      string    `gorm:"size:30;index" json:"-"`          // Username once profile is created
	Email         string    `gorm:"size:255" json:"-"`               // User's email
	Name          string    `gorm:"size:255" json:"-"`               // User's full name
	Picture       string    `gorm:"size:512" json:"-"`               // User's profile picture URL
	EmailVerified bool      `gorm:"not null;default:false" json:"-"` // Whether email is verified
	GivenName     string    `gorm:"size:100" json:"-"`               // First name
	FamilyName    string    `gorm:"size:100" json:"-"`               // Last name
	Locale        string    `gorm:"size:10" json:"-"`                // User's locale
	IPAddress     string    `gorm:"size:45" json:"-"`                // User's IP address
	UserAgent     string    `gorm:"size:255" json:"-"`               // User's browser/device info
	CreatedAt     time.Time `gorm:"not null" json:"-"`
	ExpiresAt     time.Time `gorm:"index" json:"-"`
}

// BeforeCreate hook for sessions
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.ExpiresAt.IsZero() {
		// Default session expiry using SessionDuration constant
		s.ExpiresAt = now.Add(SessionDuration)
	}
	return nil
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

