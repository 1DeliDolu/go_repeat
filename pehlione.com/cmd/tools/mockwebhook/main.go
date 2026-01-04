package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type webhookPayload struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		PaymentRef  string `json:"payment_ref"`
		RefundRef   string `json:"refund_ref"`
		AmountCents int    `json:"amount_cents"`
		Currency    string `json:"currency"`
	} `json:"data"`
}

func main() {
	url := flag.String("url", "http://localhost:8080/webhooks/mock", "Webhook URL")
	secret := flag.String("secret", os.Getenv("MOCK_WEBHOOK_SECRET"), "Webhook secret")
	eventID := flag.String("event-id", "evt_"+randomHex(8), "Event ID")
	eventType := flag.String("type", "payment.succeeded", "Event type (payment.succeeded, payment.failed, refund.succeeded, refund.failed)")
	paymentRef := flag.String("payment-ref", "pay_"+randomHex(8), "Payment ref (for payment events)")
	refundRef := flag.String("refund-ref", "", "Refund ref (for refund events)")
	amount := flag.Int("amount", 5000, "Amount in cents")
	currency := flag.String("currency", "EUR", "Currency")
	dryRun := flag.Bool("dry-run", false, "Only print signature header, don't send")

	flag.Parse()

	if *secret == "" {
		fmt.Fprintf(os.Stderr, "Error: secret not provided and MOCK_WEBHOOK_SECRET not set\n")
		os.Exit(1)
	}

	// Build payload
	payload := webhookPayload{
		ID:   *eventID,
		Type: *eventType,
	}
	payload.Data.PaymentRef = *paymentRef
	payload.Data.RefundRef = *refundRef
	payload.Data.AmountCents = *amount
	payload.Data.Currency = *currency

	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling payload: %v\n", err)
		os.Exit(1)
	}

	// Compute signature
	t := time.Now().Unix()
	sig := computeSig([]byte(*secret), t, body)

	sigHeader := fmt.Sprintf("t=%d,v1=%s", t, sig)

	fmt.Printf("X-Mock-Signature: %s\n", sigHeader)
	fmt.Printf("Body: %s\n", string(body))

	if *dryRun {
		fmt.Println("\n[DRY RUN] Not sending request")
		return
	}

	// Send webhook
	fmt.Printf("\nSending to %s...\n", *url)
	req, err := http.NewRequest("POST", *url, bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Signature", sigHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(respBody))

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}

func computeSig(secret []byte, t int64, body []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(strconv.FormatInt(t, 10)))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func randomHex(n int) string {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = "0123456789abcdef"[time.Now().UnixNano()%16]
	}
	return string(b)
}
