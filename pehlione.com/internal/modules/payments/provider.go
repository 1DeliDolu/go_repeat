package payments

import (
	"context"
	"net/http"
)

type CreatePaymentRequest struct {
	OrderID        string
	AmountCents    int
	Currency       string
	IdempotencyKey string
	ReturnURL      string
	CancelURL      string
}

type CreatePaymentResponse struct {
	ProviderRef string
	Status      string // initiated|succeeded|failed|requires_redirect
	RedirectURL string
}

type RefundRequest struct {
	OrderID        string
	PaymentID      string
	PaymentRef     string // payment.provider_ref (if available)
	AmountCents    int
	Currency       string
	IdempotencyKey string
	Reason         string
}

type RefundResponse struct {
	ProviderRef string
	Status      string // initiated|succeeded|failed
}

type WebhookEvent struct {
	EventID string
	Type    string // payment.succeeded|payment.failed|refund.succeeded|refund.failed

	PaymentRef string // provider_ref
	RefundRef  string // provider_ref

	AmountCents int
	Currency    string
}

type Provider interface {
	Name() string
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error)
	RefundPayment(ctx context.Context, req RefundRequest) (RefundResponse, error)

	// Webhook: verify signature + parse event
	VerifyAndParseWebhook(headers http.Header, body []byte) (WebhookEvent, error)
}
