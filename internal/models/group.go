package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ActivityType represents the type of activity for a group
type ActivityType string

const (
	SportActivity  ActivityType = "sport"
	SocialActivity ActivityType = "social"
	GamesActivity  ActivityType = "games"
	OtherActivity  ActivityType = "other"
)

// SkillLevel represents the required skill level for a group
type SkillLevel string

const (
	Beginner     SkillLevel = "beginner"
	Intermediate SkillLevel = "intermediate"
	Advanced     SkillLevel = "advanced"
)

// Venue represents a location using Google Maps data
type Venue struct {
	FormattedAddress string  `json:"formatted_address" binding:"required"`
	PlaceID          string  `json:"place_id" binding:"required"`
	Latitude         float64 `json:"latitude" binding:"required"`
	Longitude        float64 `json:"longitude" binding:"required"`
}
// Implement driver.Valuer and sql.Scanner
func (v Venue) Value() (driver.Value, error) {
	return json.Marshal(v)
}

func (v *Venue) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Venue: %v", value)
	}
	return json.Unmarshal(bytes, v)
}

// Group represents a group in the system
type Group struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	DateTime     time.Time    `json:"date_time"`
	Venue        Venue        `json:"venue" gorm:"type:json"`
	Cost         float64      `json:"cost"`
	SkillLevel   SkillLevel   `json:"skill_level"`
	ActivityType ActivityType `json:"activity_type"`
	MaxMembers   int          `json:"max_members"`
	Description  string       `json:"description"`
	OrganiserID  string       `json:"organiser_id"`
	Members      MemberList     `json:"members"`
}

// Member represents a user's membership status in a group
type Member struct {
	Username string `json:"username"`
	Status   string `json:"status"` // "pending", "approved", "rejected"
}
type MemberList []Member

func (m MemberList) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *MemberList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal MemberList: %v", value)
	}
	return json.Unmarshal(bytes, m)
}

// CreateGroupRequest represents the data needed to create a new group
type CreateGroupRequest struct {
	Name         string       `json:"name" binding:"required"`
	DateTime     time.Time    `json:"date_time" binding:"required"`
	Venue        Venue        `json:"venue" binding:"required"`
	Cost         float64      `json:"cost" binding:"required,min=0"`
	SkillLevel   SkillLevel   `json:"skill_level" binding:"required,oneof=beginner intermediate advanced"`
	ActivityType ActivityType `json:"activity_type" binding:"required,oneof=sport social games other"`
	MaxMembers   int          `json:"max_members" binding:"required,min=2"`
	Description  string       `json:"description" binding:"required"`
}
