package notifications

import (
	"context"
	"fmt"
	"net/smtp"
)

// EmailProvider sends notifications via SMTP.
type EmailProvider struct {
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
}

// NewEmailProvider creates an SMTP email provider.
// Returns nil (not configured) if any required field is empty.
func NewEmailProvider(host string, port int, username, password, from string, useTLS bool) *EmailProvider {
	if host == "" || from == "" {
		return nil
	}
	return &EmailProvider{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		useTLS:   useTLS,
	}
}

func (e *EmailProvider) Send(ctx context.Context, n *Notification) error {
	if e == nil {
		return fmt.Errorf("email provider not initialized")
	}

	subject := fmt.Sprintf("[Predict-A-Trade] %s: %s", n.EventType, n.Title)
	body := fmt.Sprintf("Event: %s\nSeverity: %s\n\n%s\n\nTime: %s",
		n.EventType, n.Severity, n.Message, n.CreatedAt.Format("2006-01-02 15:04:05 UTC"))

	msg := fmt.Sprintf("From: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		e.from, subject, body)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	var auth smtp.Auth
	if e.username != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	recipients := []string{n.UserAccount}
	if n.UserAccount == "" {
		return fmt.Errorf("no recipient for notification %s", n.NotificationID)
	}

	return smtp.SendMail(addr, auth, e.from, recipients, []byte(msg))
}

func (e *EmailProvider) Channel() Channel { return ChannelEmail }
func (e *EmailProvider) IsConfigured() bool {
	return e != nil && e.host != "" && e.from != ""
}
func (e *EmailProvider) Name() string { return "smtp" }
