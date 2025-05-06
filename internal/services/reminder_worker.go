package services

import (
	"groops/internal/database"
	"groops/internal/models"
	"log"
	"time"

	"gorm.io/gorm"
)

type ReminderWorker struct {
	db           *gorm.DB
	emailService *EmailService
	interval     time.Duration
}

func NewReminderWorker() *ReminderWorker {
	return &ReminderWorker{
		db:           database.GetDB(),
		emailService: NewEmailService(),
		interval:     time.Minute * 5, // Check every 5 minutes
	}
}

func (w *ReminderWorker) Start() {
	go w.run()
}

func (w *ReminderWorker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for range ticker.C {
		w.checkUpcomingEvents()
	}
}

// Check if event is within the reminder window
func isWithinReminderWindow(eventTime time.Time, currentTime time.Time, window time.Duration) bool {
	timeUntilEvent := eventTime.Sub(currentTime)
	return timeUntilEvent <= window && timeUntilEvent > (window-10*time.Minute)
}

// Check if reminders have been sent for this group already
func (w *ReminderWorker) hasReminderBeenSent(groupID string, reminderType string) bool {
	var count int64
	w.db.Model(&models.ReminderSent{}).
		Where("group_id = ? AND reminder_type = ?", groupID, reminderType).
		Count(&count)
	return count > 0
}

// Record that reminders were sent
func (w *ReminderWorker) recordReminders(groupID string, usernames []string, reminderType string) {
	now := time.Now()
	for _, username := range usernames {
		reminder := models.ReminderSent{
			GroupID:      groupID,
			Username:     username,
			ReminderType: reminderType,
			SentAt:       now,
		}
		w.db.Create(&reminder)
	}
}

func (w *ReminderWorker) checkUpcomingEvents() {
	now := time.Now()

	// Find groups with events in the future
	var groups []models.Group
	w.db.Where("date_time > ?", now).Find(&groups)

	// For each group that needs reminders
	for _, group := range groups {
		// Process 24-hour reminders
		if isWithinReminderWindow(group.DateTime, now, 24*time.Hour) {
			// Check if we already sent reminders to this group
			if !w.hasReminderBeenSent(group.ID, "24hour") {
				w.sendRemindersForGroup(group, "24hour")
			}
		}

		// Process 1-hour reminders
		if isWithinReminderWindow(group.DateTime, now, 1*time.Hour) {
			if !w.hasReminderBeenSent(group.ID, "1hour") {
				w.sendRemindersForGroup(group, "1hour")
			}
		}
	}
}

func (w *ReminderWorker) sendRemindersForGroup(group models.Group, reminderType string) {
	// Get all approved members for this group
	var members []models.GroupMember
	w.db.Where("group_id = ? AND status = ?", group.ID, "approved").Find(&members)

	if len(members) == 0 {
		return
	}

	// Get member usernames
	var memberUsernames []string
	for _, m := range members {
		memberUsernames = append(memberUsernames, m.Username)
	}

	// Get all member accounts in one query
	var accounts []models.Account
	w.db.Where("username IN ?", memberUsernames).Find(&accounts)

	if len(accounts) == 0 {
		return
	}

	// Send batch email to all members
	err := w.emailService.SendEventReminderToGroup(group, accounts, reminderType)
	if err != nil {
		log.Printf("Failed to send %s reminders for group %s: %v", reminderType, group.ID, err)
		return
	}

	// Record that reminders were sent
	w.recordReminders(group.ID, memberUsernames, reminderType)
	log.Printf("Sent %s reminders to %d members for group %s", reminderType, len(accounts), group.ID)
}
