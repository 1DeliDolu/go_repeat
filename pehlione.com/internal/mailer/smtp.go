package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"pehlione.com/app/internal/config"
)

type SMTPMailer struct {
	cfg          config.SMTPConfig
	dialTimeout  time.Duration
	writeTimeout time.Duration

	// Message-ID domain için (host yoksa fallback)
	messageIDDomain string
}

func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer {
	domain := cfg.Host
	if domain == "" {
		domain = "local"
	}
	return &SMTPMailer{
		cfg:             cfg,
		dialTimeout:     5 * time.Second,
		writeTimeout:    10 * time.Second,
		messageIDDomain: domain,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, e Email) error {
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)

	raw, err := buildMIMEMessage(e, m.messageIDDomain)
	if err != nil {
		return err
	}

	// ctx ile Dial
	dialer := &net.Dialer{Timeout: m.dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial failed: %w", err)
	}
	defer conn.Close()

	// TLSMode=tls ise baştan TLS socket
	if strings.EqualFold(m.cfg.TLSMode, "tls") {
		tlsCfg := &tls.Config{
			ServerName:         m.cfg.Host,
			InsecureSkipVerify: m.cfg.SkipVerifyTLS,
		}
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("smtp tls handshake failed: %w", err)
		}
		conn = tlsConn
	}

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp new client failed: %w", err)
	}
	defer c.Quit()

	// STARTTLS
	if strings.EqualFold(m.cfg.TLSMode, "starttls") {
		if ok, _ := c.Extension("STARTTLS"); ok {
			tlsCfg := &tls.Config{
				ServerName:         m.cfg.Host,
				InsecureSkipVerify: m.cfg.SkipVerifyTLS,
			}
			if err := c.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("smtp starttls failed: %w", err)
			}
		} else {
			return fmt.Errorf("smtp starttls not supported by server")
		}
	}

	// AUTH (MailHog default: auth yok; user/pass boşsa geç)
	if m.cfg.User != "" && m.cfg.Pass != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth failed: %w", err)
			}
		}
	}

	// MAIL FROM / RCPT TO
	if err := c.Mail(e.From); err != nil {
		return fmt.Errorf("smtp mail from failed: %w", err)
	}
	for _, rcpt := range e.AllRecipients() {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt failed (%s): %w", rcpt, err)
		}
	}

	// DATA
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data failed: %w", err)
	}

	// Basit write timeout
	_ = conn.SetWriteDeadline(time.Now().Add(m.writeTimeout))
	if _, err := w.Write([]byte(raw)); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close failed: %w", err)
	}

	return nil
}
