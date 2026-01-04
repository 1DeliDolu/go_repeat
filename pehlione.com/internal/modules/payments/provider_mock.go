package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidSignature = errors.New("invalid webhook signature")
var ErrTimestampOutOfRange = errors.New("webhook timestamp out of range")

type MockProvider struct {
	WebhookSecret []byte
}

func NewMockProvider(secret string) MockProvider {
	if secret == "" {
		secret = "dev_secret_change_me"
	}
	return MockProvider{WebhookSecret: []byte(secret)}
}

func (MockProvider) Name() string { return "mock" }

func (MockProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error) {
	_ = ctx
	_ = req
	// Async: return initiated; webhook later provides success/failure
	return CreatePaymentResponse{
		ProviderRef: uuid.NewString(),
		Status:      StatusInitiated,
	}, nil
}

func (MockProvider) RefundPayment(ctx context.Context, req RefundRequest) (RefundResponse, error) {
	_ = ctx
	_ = req
	// Async: return initiated; webhook later provides success/failure
	return RefundResponse{
		ProviderRef: uuid.NewString(),
		Status:      StatusInitiated,
	}, nil
}

type mockWebhookPayload struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		PaymentRef  string `json:"payment_ref"`
		RefundRef   string `json:"refund_ref"`
		AmountCents int    `json:"amount_cents"`
		Currency    string `json:"currency"`
	} `json:"data"`
}

// Header: X-Mock-Signature: t=<unix>,v1=<hex_hmac>
// Signature: HMAC_SHA256(secret, "<t>.<raw_body>") (hex encode)
func (p MockProvider) VerifyAndParseWebhook(headers http.Header, body []byte) (WebhookEvent, error) {
	raw := strings.TrimSpace(headers.Get("X-Mock-Signature"))
	if raw == "" {
		return WebhookEvent{}, ErrInvalidSignature
	}

	t, sigs, ok := parseStripeLikeSigHeader(raw)
	if !ok || t <= 0 || len(sigs) == 0 {
		return WebhookEvent{}, ErrInvalidSignature
	}

	tol := toleranceSeconds()
	now := time.Now().Unix()

	// replay/clock-skew guard:
	// - çok eski: reject
	// - gelecekte çok ileri: reject
	if t < now-int64(tol) || t > now+int64(tol) {
		return WebhookEvent{}, ErrTimestampOutOfRange
	}

	expected := computeSigHex(p.WebhookSecret, t, body)

	// multiple v1 signatures support (Stripe benzeri)
	matched := false
	expBytes, _ := hex.DecodeString(expected)
	for _, s := range sigs {
		gotBytes, err := hex.DecodeString(s)
		if err != nil {
			continue
		}
		if hmac.Equal(expBytes, gotBytes) {
			matched = true
			break
		}
	}
	if !matched {
		return WebhookEvent{}, ErrInvalidSignature
	}

	// payload parse
	var pl mockWebhookPayload
	if err := json.Unmarshal(body, &pl); err != nil {
		return WebhookEvent{}, err
	}

	return WebhookEvent{
		EventID:     pl.ID,
		Type:        pl.Type,
		PaymentRef:  pl.Data.PaymentRef,
		RefundRef:   pl.Data.RefundRef,
		AmountCents: pl.Data.AmountCents,
		Currency:    pl.Data.Currency,
	}, nil
}

// parseStripeLikeSigHeader parses "t=1730000000,v1=abc,v1=def"
func parseStripeLikeSigHeader(h string) (ts int64, v1 []string, ok bool) {
	parts := strings.Split(h, ",")
	var tStr string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch k {
		case "t":
			tStr = val
		case "v1":
			if val != "" {
				v1 = append(v1, val)
			}
		}
	}
	if tStr == "" || len(v1) == 0 {
		return 0, nil, false
	}
	t, err := strconv.ParseInt(tStr, 10, 64)
	if err != nil || t <= 0 {
		return 0, nil, false
	}
	return t, v1, true
}

func computeSigHex(secret []byte, t int64, body []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(strconv.FormatInt(t, 10)))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func toleranceSeconds() int {
	v := strings.TrimSpace(os.Getenv("MOCK_WEBHOOK_TOLERANCE_SECONDS"))
	if v == "" {
		return 300
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 60 || n > 3600 {
		return 300
	}
	return n
}
