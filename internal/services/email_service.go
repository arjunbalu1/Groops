package services

import (
	"fmt"
	"groops/internal/models"
	"os"

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

// SendEventReminderToGroup sends event reminders to all members in a group
func (s *EmailService) SendEventReminderToGroup(group models.Group, members []models.Account, reminderType string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	message := mail.NewV3Mail()
	message.SetFrom(from)

	// Set template ID - you'll need to create this in SendGrid
	// For now, we'll use dynamic content without a template
	if reminderType == "24hour" {
		message.Subject = fmt.Sprintf("Reminder: %s is tomorrow", group.Name)
	} else {
		message.Subject = fmt.Sprintf("Reminder: %s starts in 1 hour", group.Name)
	}

	// Add each member with personalization
	for _, member := range members {
		personalization := mail.NewPersonalization()
		to := mail.NewEmail(member.Username, member.Email)
		personalization.AddTos(to)

		// Add custom fields for dynamic content
		personalization.SetDynamicTemplateData("username", member.Username)
		personalization.SetDynamicTemplateData("group_name", group.Name)
		personalization.SetDynamicTemplateData("event_time", group.DateTime.Format("Mon Jan 2, 3:04 PM"))
		personalization.SetDynamicTemplateData("location_name", group.Location.Name)

		message.AddPersonalizations(personalization)
	}

	// For dynamic content without a template
	// You should really create a template in SendGrid and use the template ID above
	content := mail.NewContent("text/html", "<p>Hello {{username}},</p><p>Your event <strong>{{group_name}}</strong> is coming up soon at {{event_time}} at {{location_name}}.</p><p>Don't miss it!</p>")
	message.AddContent(content)

	// Single API call for the entire group
	response, err := s.client.Send(message)
	if err != nil {
		return err
	}

	if response.StatusCode >= 400 {
		return fmt.Errorf("failed to send emails: %d", response.StatusCode)
	}

	return nil
}
