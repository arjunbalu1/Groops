package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ActivityType represents the type of activity for a group
// Now accepts any non-empty string value
type ActivityType string

// SkillLevel represents the required skill level for a group
type SkillLevel string

const (
	Beginner     SkillLevel = "beginner"
	Intermediate SkillLevel = "intermediate"
	Advanced     SkillLevel = "advanced"
)

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
	ID           string        `gorm:"primaryKey;size:50;not null" json:"id"`
	Name         string        `gorm:"index;size:100;not null" json:"name"`
	DateTime     time.Time     `gorm:"index;not null" json:"date_time"`
	Location     Location      `gorm:"type:jsonb;not null" json:"location"`
	Cost         float64       `gorm:"type:decimal(10,2);not null;default:0.0" json:"cost"`
	SkillLevel   *string       `gorm:"type:varchar(20);index" json:"skill_level,omitempty"`
	ActivityType string        `gorm:"type:varchar(50);index;not null" json:"activity_type"`
	MaxMembers   int           `gorm:"type:integer;not null;default:10" json:"max_members"`
	Description  string        `gorm:"type:text;not null;size:1000" json:"description"`
	OrganiserID  string        `gorm:"index;size:30;not null" json:"organiser_id"`
	Members      []GroupMember `gorm:"foreignKey:GroupID" json:"members"`
	CreatedAt    time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time     `gorm:"not null" json:"updated_at"`
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
	Name         string    `json:"name" binding:"required"`
	DateTime     time.Time `json:"date_time" binding:"required"`
	Location     Location  `json:"location" binding:"required"`
	Cost         float64   `json:"cost"`
	SkillLevel   *string   `json:"skill_level,omitempty"`
	ActivityType string    `json:"activity_type" binding:"required"`
	MaxMembers   int       `json:"max_members" binding:"required,min=2,max=50"`
	Description  string    `json:"description" binding:"required,max=1000"`
}
