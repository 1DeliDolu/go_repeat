package mailer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"strings"
	"time"
)

func formatAddress(name, addr string) string {
	// RFC2047: non-ascii isimleri encode eder
	if name == "" {
		return addr
	}
	encoded := mime.QEncoding.Encode("utf-8", name)
	return fmt.Sprintf("%s <%s>", encoded, addr)
}

func encodeSubject(subject string) string {
	// Subject non-ascii ise RFC2047 ile encode edilir
	return mime.QEncoding.Encode("utf-8", subject)
}

func newMessageID(domain string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(b), domain)
}

func buildMIMEMessage(e Email, messageIDDomain string) (string, error) {
	if len(e.To) == 0 {
		return "", fmt.Errorf("mailer: at least one recipient required")
	}
	if e.From == "" {
		return "", fmt.Errorf("mailer: from address required")
	}
	if e.Subject == "" {
		return "", fmt.Errorf("mailer: subject required")
	}
	if e.TextBody == "" && e.HTMLBody == "" {
		return "", fmt.Errorf("mailer: textBody or htmlBody required")
	}

	var b strings.Builder

	// Standard headers
	b.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	b.WriteString(fmt.Sprintf("Message-ID: %s\r\n", newMessageID(messageIDDomain)))
	b.WriteString(fmt.Sprintf("From: %s\r\n", formatAddress(e.FromName, e.From)))
	b.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.To, ", ")))
	if len(e.Cc) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(e.Cc, ", ")))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(e.Subject)))
	b.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for k, v := range e.Headers {
		if k == "" || v == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Body
	if e.TextBody != "" && e.HTMLBody != "" {
		// multipart/alternative
		boundary := randomBoundary()
		b.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		b.WriteString("\r\n")

		// text part
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		b.WriteString("\r\n")
		b.WriteString(e.TextBody)
		if !strings.HasSuffix(e.TextBody, "\n") {
			b.WriteString("\r\n")
		}

		// html part
		b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		b.WriteString("\r\n")
		b.WriteString(e.HTMLBody)
		if !strings.HasSuffix(e.HTMLBody, "\n") {
			b.WriteString("\r\n")
		}

		b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		return b.String(), nil
	}

	if e.HTMLBody != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 7bit\r\n")
		b.WriteString("\r\n")
		b.WriteString(e.HTMLBody)
		if !strings.HasSuffix(e.HTMLBody, "\n") {
			b.WriteString("\r\n")
		}
		return b.String(), nil
	}

	// text-only
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(e.TextBody)
	if !strings.HasSuffix(e.TextBody, "\n") {
		b.WriteString("\r\n")
	}
	return b.String(), nil
}

func randomBoundary() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "alt-" + hex.EncodeToString(b)
}
