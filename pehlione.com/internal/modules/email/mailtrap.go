package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type MailtrapProvider struct {
	apiURL string
	apiKey string
}

type MailtrapPayload struct {
	From     PersonInfo   `json:"from"`
	To       []PersonInfo `json:"to"`
	Subject  string       `json:"subject"`
	Text     string       `json:"text,omitempty"`
	HTML     string       `json:"html,omitempty"`
	Category string       `json:"category,omitempty"`
}

type PersonInfo struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

func NewMailtrapProvider() *MailtrapProvider {
	return &MailtrapProvider{
		apiURL: os.Getenv("MAILTRAP_API_URL"),   // e.g., "https://sandbox.api.mailtrap.io/api/send/3939647"
		apiKey: os.Getenv("MAILTRAP_API_TOKEN"), // Bearer token
	}
}

func (m *MailtrapProvider) SendEmail(to string, toName string, subject string, htmlBody string, textBody string) error {
	if m.apiURL == "" || m.apiKey == "" {
		return fmt.Errorf("mailtrap credentials not configured")
	}

	payload := MailtrapPayload{
		From: PersonInfo{
			Email: os.Getenv("EMAIL_FROM"),
			Name:  os.Getenv("EMAIL_FROM_NAME"),
		},
		To: []PersonInfo{
			{
				Email: to,
				Name:  toName,
			},
		},
		Subject:  subject,
		HTML:     htmlBody,
		Text:     textBody,
		Category: "Transactional",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", m.apiURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", m.apiKey))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return fmt.Errorf("mailtrap API error: %d", res.StatusCode)
	}

	return nil
}
