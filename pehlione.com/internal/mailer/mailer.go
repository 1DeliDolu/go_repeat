package mailer

import "context"

type Service interface {
	Send(ctx context.Context, e Email) error
}

type Email struct {
	FromName string // opsiyonel: "My Shop"
	From     string // zorunlu: "no-reply@local.test"

	To  []string
	Cc  []string
	Bcc []string

	Subject string

	TextBody string
	HTMLBody string

	Headers map[string]string // opsiyonel ekstra header
}

func (e Email) AllRecipients() []string {
	out := make([]string, 0, len(e.To)+len(e.Cc)+len(e.Bcc))
	out = append(out, e.To...)
	out = append(out, e.Cc...)
	out = append(out, e.Bcc...)
	return out
}
