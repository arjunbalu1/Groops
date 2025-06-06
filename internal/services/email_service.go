package services

import (
	"fmt"
	"groops/internal/models"
	"os"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type EmailService struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

func NewEmailService() *EmailService {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	fromEmail := os.Getenv("SENDGRID_NOTIFICATIONS_FROM_EMAIL")
	fromName := os.Getenv("SENDGRID_FROM_NAME")

	client := sendgrid.NewSendClient(apiKey)

	return &EmailService{
		client:    client,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// convertToIST converts UTC time to IST (Indian Standard Time)
func convertToIST(utcTime time.Time) time.Time {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		// Fallback: manually add 5 hours 30 minutes
		return utcTime.Add(5*time.Hour + 30*time.Minute)
	}
	return utcTime.In(ist)
}

// SendWelcomeEmail sends a welcome email to users who register a username
func (s *EmailService) SendWelcomeEmail(userEmail, userName string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(userName, userEmail)
	subject := "Welcome to Groops!"
	plainContent := fmt.Sprintf("Hello %s, Welcome to Groops! We're excited to have you join our community. Start exploring groups and activities now!", userName)
	htmlContent := fmt.Sprintf("<p>Hello <strong>%s</strong>,</p><p>Welcome to <strong>Groops</strong>! We're excited to have you join our community.</p><p>Start exploring groups and activities now!</p>", userName)

	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)
	_, err := s.client.Send(message)
	return err
}

// SendAdminOAuthNotification notifies admin when someone completes OAuth login
func (s *EmailService) SendAdminOAuthNotification(userEmail, userName string) error {
	adminEmail := os.Getenv("ADMIN_NOTIFICATION_EMAIL")
	if adminEmail == "" {
		return fmt.Errorf("ADMIN_NOTIFICATION_EMAIL environment variable not set")
	}

	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail("Admin", adminEmail)
	subject := "New OAuth Login on Groops"
	plainContent := fmt.Sprintf("A new user has completed OAuth login: %s (%s)", userName, userEmail)
	htmlContent := fmt.Sprintf("<p>A new user has completed OAuth login:</p><p><strong>Name:</strong> %s</p><p><strong>Email:</strong> %s</p>", userName, userEmail)

	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)
	_, err := s.client.Send(message)
	return err
}

// SendJoinRequestEmail notifies group owner of new join request
func (s *EmailService) SendJoinRequestEmail(ownerEmail, ownerName, requesterName, groupName string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(ownerName, ownerEmail)
	subject := fmt.Sprintf("New Join Request for %s", groupName)
	plainContent := fmt.Sprintf("%s has requested to join your group '%s'", requesterName, groupName)
	htmlContent := fmt.Sprintf("<p>%s has requested to join your group '<strong>%s</strong>'</p>", requesterName, groupName)

	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)
	_, err := s.client.Send(message)
	return err
}

// SendJoinApprovalEmail notifies user their request was approved
func (s *EmailService) SendJoinApprovalEmail(userEmail, userName, groupName string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(userName, userEmail)
	subject := fmt.Sprintf("You're in! Join request for %s approved", groupName)
	plainContent := fmt.Sprintf("Your request to join '%s' has been approved!", groupName)
	htmlContent := fmt.Sprintf("<p>Good news! Your request to join '<strong>%s</strong>' has been approved!</p>", groupName)

	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)
	_, err := s.client.Send(message)
	return err
}

// SendMemberRemovalEmail notifies user they've been removed from a group
func (s *EmailService) SendMemberRemovalEmail(userEmail, userName, groupName string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(userName, userEmail)
	subject := fmt.Sprintf("You have been removed from %s", groupName)
	plainContent := fmt.Sprintf("You have been removed from the group '%s'", groupName)
	htmlContent := fmt.Sprintf("<p>You have been removed from the group '<strong>%s</strong>'</p>", groupName)

	message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)
	_, err := s.client.Send(message)
	return err
}

// SendEventReminderToGroup sends event reminders to all members in a group
func (s *EmailService) SendEventReminderToGroup(group models.Group, members []models.Account, reminderType string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)

	// Convert UTC time to IST for display
	istTime := convertToIST(group.DateTime)
	timeStr := istTime.Format("Mon Jan 2, 3:04 PM") + " IST"

	// Simple subject based on reminder type
	subject := ""
	if reminderType == "24hour" {
		subject = fmt.Sprintf("Reminder: %s is tomorrow", group.Name)
	} else {
		subject = fmt.Sprintf("Reminder: %s starts in 1 hour", group.Name)
	}

	// Send individual emails to each member
	for _, member := range members {
		to := mail.NewEmail(member.Username, member.Email)

		// Use direct string formatting with IST time
		plainContent := fmt.Sprintf("Hello %s, Your event %s is coming up soon at %s at %s. Don't miss it!",
			member.Username, group.Name, timeStr, group.Location.Name)

		htmlContent := fmt.Sprintf("<p>Hello %s,</p><p>Your event <strong>%s</strong> is coming up soon at %s at %s.</p><p>Don't miss it!</p>",
			member.Username, group.Name, timeStr, group.Location.Name)

		// Create a simple email without template variables
		message := mail.NewSingleEmail(from, subject, to, plainContent, htmlContent)

		// Send email
		response, err := s.client.Send(message)
		if err != nil {
			return err
		}

		if response.StatusCode >= 400 {
			return fmt.Errorf("failed to send email to %s: %d", member.Email, response.StatusCode)
		}
	}

	return nil
}
