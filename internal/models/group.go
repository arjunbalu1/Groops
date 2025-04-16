package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
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

// Implement driver.Valuer and sql.Scanner for JSONB storage
func (v Venue) Value() (driver.Value, error) {
	return json.Marshal(v)
}

func (v *Venue) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Venue: %v", value)
	}
	return json.Unmarshal(bytes, v)
}

// Member represents a user's membership status in a group
type GroupMember struct {
	GroupID   string    `gorm:"primaryKey;size:50" json:"group_id"`
	Username  string    `gorm:"primaryKey;size:30" json:"username"`
	Status    string    `gorm:"size:20;not null;default:'pending'" json:"status"` // pending, approved, rejected
	JoinedAt  time.Time `gorm:"not null" json:"joined_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// Group represents a group in the system
type Group struct {
	ID           string         `gorm:"primaryKey;size:50;not null" json:"id"`
	Name         string         `gorm:"index;size:100;not null" json:"name"`
	DateTime     time.Time      `gorm:"index;not null" json:"date_time"`
	Venue        Venue          `gorm:"type:jsonb;not null" json:"venue"`
	Cost         float64        `gorm:"type:decimal(10,2);not null;default:0.0" json:"cost"`
	SkillLevel   SkillLevel     `gorm:"type:varchar(20);not null;default:'beginner'" json:"skill_level"`
	ActivityType ActivityType   `gorm:"type:varchar(20);index;not null" json:"activity_type"`
	MaxMembers   int            `gorm:"type:integer;not null;default:10" json:"max_members"`
	Description  string         `gorm:"type:text;not null" json:"description"`
	OrganiserID  string         `gorm:"index;size:30;not null" json:"organiser_id"`
	Members      []GroupMember  `gorm:"foreignKey:GroupID" json:"members"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook is called before creating a new group
func (g *Group) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	if g.UpdatedAt.IsZero() {
		g.UpdatedAt = now
	}
	if g.ID == "" {
		g.ID = fmt.Sprintf("%s-%s", g.OrganiserID, now.UTC().Format("20060102150405"))
	}
	return nil
}

// BeforeSave hook is called before saving the group
func (g *Group) BeforeSave(tx *gorm.DB) error {
	g.UpdatedAt = time.Now()
	return nil
}

// TableName specifies the table name for the Group model
func (Group) TableName() string {
	return "group"
}

// TableName specifies the table name for the GroupMember model
func (GroupMember) TableName() string {
	return "group_member"
}

// BeforeCreate hook is called before creating a new group member
func (gm *GroupMember) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if gm.JoinedAt.IsZero() {
		gm.JoinedAt = now
	}
	if gm.UpdatedAt.IsZero() {
		gm.UpdatedAt = now
	}
	return nil
}

// BeforeSave hook is called before saving the group member
func (gm *GroupMember) BeforeSave(tx *gorm.DB) error {
	gm.UpdatedAt = time.Now()
	return nil
}

// CreateGroupRequest represents the data needed to create a new group
type CreateGroupRequest struct {
	Name              string       `json:"name" binding:"required"`
	DateTime          time.Time    `json:"date_time" binding:"required"`
	Venue             Venue        `json:"venue" binding:"required"`
	Cost              float64      `json:"cost" binding:"required,min=0"`
	SkillLevel        SkillLevel   `json:"skill_level" binding:"required,oneof=beginner intermediate advanced"`
	ActivityType      ActivityType `json:"activity_type" binding:"required,oneof=sport social games other"`
	MaxMembers        int          `json:"max_members" binding:"required,min=2"`
	Description       string       `json:"description" binding:"required"`
	OrganizerUsername string       `json:"organizer_username" binding:"required"`
}
