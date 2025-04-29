package models

import (
	"time"

	"gorm.io/gorm"
)

// SessionDuration is the length of time a session remains valid
const SessionDuration = time.Hour * 24 * 7 // 1 week

// Session represents a user session with OAuth tokens
type Session struct {
	ID          string    `gorm:"primaryKey;size:64" json:"-"`
	UserID      string    `gorm:"size:128;index" json:"-"` // Google ID
	Username    string    `gorm:"size:30;index" json:"-"`  // Username once profile is created
	AccessToken string    `gorm:"type:text" json:"-"`
	TokenExpiry time.Time `gorm:"index" json:"-"`
	CreatedAt   time.Time `gorm:"not null" json:"-"`
	ExpiresAt   time.Time `gorm:"index" json:"-"`
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

// NeedsTokenRefresh checks if the access token needs to be refreshed
func (s *Session) NeedsTokenRefresh() bool {
	// If expiry is zero time, it needs refresh
	if s.TokenExpiry.IsZero() {
		return true
	}

	// Refresh 5 minutes before expiry to avoid edge cases
	return time.Now().Add(time.Minute * 5).After(s.TokenExpiry)
}

// HasActiveUser returns true if the session is associated with a registered user
func (s *Session) HasActiveUser() bool {
	return s.Username != ""
}
