package models

import "time"

// ReminderSent tracks which reminders have been sent to avoid duplicates
type ReminderSent struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	GroupID      string    `gorm:"size:50;not null;index" json:"group_id"`
	Username     string    `gorm:"size:30;not null;index" json:"username"`
	ReminderType string    `gorm:"size:10;not null" json:"reminder_type"` // "24hour" or "1hour"
	SentAt       time.Time `gorm:"not null" json:"sent_at"`
}
