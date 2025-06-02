package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Message represents a chat message in a group
type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	GroupID   string         `gorm:"size:50;not null;index:idx_messages_group_created" json:"group_id"`
	Username  string         `gorm:"size:30;not null;index" json:"username"`
	Content   string         `gorm:"type:text;not null;size:1000" json:"content"`
	ReadBy    datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"read_by"`
	CreatedAt time.Time      `gorm:"not null;index:idx_messages_group_created" json:"created_at"`

	// Relationships
	Group Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// BeforeCreate hook is called before creating a new message
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	return nil
}

// SendMessageRequest represents the data needed to send a message
type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=1000"`
}
