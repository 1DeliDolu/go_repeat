package email

import (
	"context"

	"pehlione.com/app/internal/mailer"
)

// MailerAdapter wraps mailer.SMTPMailer to implement email.Sender interface
type MailerAdapter struct {
	mailer   *mailer.SMTPMailer
	fromAddr string
	fromName string
}

// NewMailerAdapter creates a new adapter from mailer.SMTPMailer
func NewMailerAdapter(m *mailer.SMTPMailer, fromAddr, fromName string) *MailerAdapter {
	return &MailerAdapter{
		mailer:   m,
		fromAddr: fromAddr,
		fromName: fromName,
	}
}

// Send implements the email.Sender interface
func (a *MailerAdapter) Send(ctx context.Context, m Message) error {
	email := mailer.Email{
		From:     a.fromAddr,
		FromName: a.fromName,
		To:       []string{m.To},
		Subject:  m.Subject,
		TextBody: m.Text,
		HTMLBody: m.HTML,
	}
	return a.mailer.Send(ctx, email)
}
