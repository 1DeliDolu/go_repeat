package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPCfg struct {
	Host   string
	Port   int
	User   string
	Pass   string
	From   string
	UseTLS bool
}

type SMTPSender struct {
	cfg SMTPCfg
}

func NewSMTPSender(cfg SMTPCfg) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, m Message) error {
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Pass, s.cfg.Host)
	}

	body := buildMIME(s.cfg.From, m.To, m.Subject, m.Text, m.HTML)

	// No TLS: plain connection (e.g., MailHog development)
	if !s.cfg.UseTLS {
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			return err
		}
		defer c.Close()

		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}

		if err := c.Mail(s.cfg.From); err != nil {
			return err
		}
		if err := c.Rcpt(m.To); err != nil {
			return err
		}

		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			_ = w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}

		return c.Quit()
	}

	// For port 587, use STARTTLS (upgrade plain connection)
	// For other ports with TLS, use direct TLS connection
	if s.cfg.Port == 587 || s.cfg.Port == 25 {
		// STARTTLS flow
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			return err
		}
		defer c.Close()

		// Issue STARTTLS command
		if err := c.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return err
		}

		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}

		if err := c.Mail(s.cfg.From); err != nil {
			return err
		}
		if err := c.Rcpt(m.To); err != nil {
			return err
		}

		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			_ = w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}

		return c.Quit()
	}

	// For direct TLS connections (port 465, 2525, etc.)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{ServerName: s.cfg.Host})
	c, err := smtp.NewClient(tlsConn, s.cfg.Host)
	if err != nil {
		return err
	}
	defer c.Close()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}

	if err := c.Mail(s.cfg.From); err != nil {
		return err
	}
	if err := c.Rcpt(m.To); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(body)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return c.Quit()
}

func buildMIME(from, to, subject, text, html string) string {
	subject = strings.ReplaceAll(subject, "\n", " ")
	subject = strings.ReplaceAll(subject, "\r", " ")

	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: multipart/alternative; boundary=BOUNDARY\r\n")
	sb.WriteString("\r\n--BOUNDARY\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	sb.WriteString(text + "\r\n")
	sb.WriteString("\r\n--BOUNDARY\r\n")
	sb.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
	sb.WriteString(html + "\r\n")
	sb.WriteString("\r\n--BOUNDARY--\r\n")
	return sb.String()
}
