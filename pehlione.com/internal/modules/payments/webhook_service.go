package payments

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pehlione.com/app/internal/modules/orders"
)

type ProviderEvent struct {
	ID          string         `gorm:"type:char(36);primaryKey"`
	Provider    string         `gorm:"type:varchar(64);not null;uniqueIndex:ux_provider_events_provider_event,priority:1"`
	EventID     string         `gorm:"type:varchar(128);not null;uniqueIndex:ux_provider_events_provider_event,priority:2"`
	EventType   string         `gorm:"type:varchar(64);not null"`
	PayloadJSON datatypes.JSON `gorm:"type:json;not null"`

	ReceivedAt   time.Time  `gorm:"type:datetime(3);not null"`
	ProcessedAt  *time.Time `gorm:"type:datetime(3)"`
	ProcessError *string    `gorm:"type:varchar(255)"`
}

func (ProviderEvent) TableName() string { return "provider_events" }

type WebhookService struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{db: db, logger: slog.Default()}
}

func (s *WebhookService) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

func (s *WebhookService) Handle(ctx context.Context, providerName string, ev WebhookEvent, rawBody []byte) error {
	// event payload'ı persist etmek için:
	payload, _ := json.RawMessage(rawBody).MarshalJSON()

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		pe := ProviderEvent{
			ID:           uuid.NewString(),
			Provider:     providerName,
			EventID:      ev.EventID,
			EventType:    ev.Type,
			PayloadJSON:  datatypes.JSON(payload),
			ReceivedAt:   now,
			ProcessedAt:  nil,
			ProcessError: nil,
		}

		// dedupe: unique(provider,event_id)
		if err := tx.WithContext(ctx).Create(&pe).Error; err != nil {
			if isDup(err) {
				// event daha önce alındı => 200 OK dönmek için nil
				s.logger.InfoContext(ctx, "webhook event deduplicated", "provider", providerName, "event_id", ev.EventID, "type", ev.Type)
				return nil
			}
			s.logger.ErrorContext(ctx, "failed to persist provider event", "provider", providerName, "event_id", ev.EventID, "err", err)
			return err
		}

		// apply
		var applyErr error
		switch ev.Type {
		case "payment.succeeded":
			applyErr = s.applyPaymentSucceeded(ctx, tx, providerName, ev)
		case "payment.failed":
			applyErr = s.applyPaymentFailed(ctx, tx, providerName, ev)
		case "refund.succeeded":
			applyErr = s.applyRefundSucceeded(ctx, tx, providerName, ev)
		case "refund.failed":
			applyErr = s.applyRefundFailed(ctx, tx, providerName, ev)
		default:
			applyErr = errors.New("unknown webhook event type")
		}

		if applyErr != nil {
			msg := truncate(applyErr.Error(), 250)
			if err := tx.WithContext(ctx).Model(&ProviderEvent{}).
				Where("id = ?", pe.ID).
				Updates(map[string]any{"process_error": msg}).Error; err != nil {
				return err
			}
			// Log apply error for debugging
			s.logger.ErrorContext(ctx, "webhook event apply failed", "provider", providerName, "event_id", ev.EventID, "type", ev.Type, "error", msg)
			// 500 dönmek için error propagate (provider retry edebilsin)
			return applyErr
		}

		processed := now
		if err := tx.WithContext(ctx).Model(&ProviderEvent{}).
			Where("id = ?", pe.ID).
			Updates(map[string]any{"processed_at": &processed, "process_error": nil}).Error; err != nil {
			return err
		}

		s.logger.InfoContext(ctx, "webhook event processed successfully", "provider", providerName, "event_id", ev.EventID, "type", ev.Type)
		return nil
	})
}

func (s *WebhookService) applyPaymentSucceeded(ctx context.Context, tx *gorm.DB, provider string, ev WebhookEvent) error {
	if ev.PaymentRef == "" {
		return errors.New("missing payment_ref")
	}

	var p Payment
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&p, "provider = ? AND provider_ref = ?", provider, ev.PaymentRef).Error; err != nil {
		return err // bulunamazsa retry
	}

	// idempotent
	if p.Status == StatusSucceeded {
		return nil
	}

	now := time.Now()
	if err := tx.WithContext(ctx).Model(&Payment{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{
			"status":        StatusSucceeded,
			"error_message": nil,
			"updated_at":    now,
		}).Error; err != nil {
		return err
	}

	// order -> paid (created ise)
	paidAt := now
	if err := tx.WithContext(ctx).Model(&orders.Order{}).
		Where("id = ? AND status = 'created'", p.OrderID).
		Updates(map[string]any{
			"status":     "paid",
			"paid_at":    &paidAt,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	// ledger (payment_succeeded)
	return ensureFinancialEntry(ctx, tx, orders.FinancialEntry{
		ID:          uuid.NewString(),
		OrderID:     p.OrderID,
		Event:       "payment_succeeded",
		AmountCents: p.AmountCents,
		Currency:    p.Currency,
		RefType:     "payment",
		RefID:       p.ID,
		CreatedAt:   now,
	})
}

func (s *WebhookService) applyPaymentFailed(ctx context.Context, tx *gorm.DB, provider string, ev WebhookEvent) error {
	if ev.PaymentRef == "" {
		return errors.New("missing payment_ref")
	}

	var p Payment
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&p, "provider = ? AND provider_ref = ?", provider, ev.PaymentRef).Error; err != nil {
		return err
	}
	if p.Status == StatusFailed {
		return nil
	}

	now := time.Now()
	return tx.WithContext(ctx).Model(&Payment{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{
			"status":        StatusFailed,
			"error_message": "provider webhook: failed",
			"updated_at":    now,
		}).Error
}

func (s *WebhookService) applyRefundSucceeded(ctx context.Context, tx *gorm.DB, provider string, ev WebhookEvent) error {
	if ev.RefundRef == "" {
		return errors.New("missing refund_ref")
	}

	var r Refund
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&r, "provider = ? AND provider_ref = ?", provider, ev.RefundRef).Error; err != nil {
		return err
	}
	if r.Status == StatusSucceeded {
		return nil
	}

	now := time.Now()
	if err := tx.WithContext(ctx).Model(&Refund{}).
		Where("id = ?", r.ID).
		Updates(map[string]any{
			"status":        StatusSucceeded,
			"error_message": nil,
			"updated_at":    now,
		}).Error; err != nil {
		return err
	}

	// order refunded totals
	var o orders.Order
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&o, "id = ?", r.OrderID).Error; err != nil {
		return err
	}

	newRefunded := o.RefundedCents + r.AmountCents
	newStatus := o.Status
	var refundedAt *time.Time

	if newRefunded >= o.TotalCents {
		newRefunded = o.TotalCents
		newStatus = "refunded"
		t := now
		refundedAt = &t
	} else {
		newStatus = "partially_refunded"
	}

	if err := tx.WithContext(ctx).Model(&orders.Order{}).
		Where("id = ?", o.ID).
		Updates(map[string]any{
			"refunded_cents": newRefunded,
			"status":         newStatus,
			"refunded_at":    refundedAt,
			"updated_at":     now,
		}).Error; err != nil {
		return err
	}

	// ledger: refund_succeeded (-)
	return ensureFinancialEntry(ctx, tx, orders.FinancialEntry{
		ID:          uuid.NewString(),
		OrderID:     r.OrderID,
		Event:       "refund_succeeded",
		AmountCents: -r.AmountCents,
		Currency:    r.Currency,
		RefType:     "refund",
		RefID:       r.ID,
		CreatedAt:   now,
	})
}

func (s *WebhookService) applyRefundFailed(ctx context.Context, tx *gorm.DB, provider string, ev WebhookEvent) error {
	if ev.RefundRef == "" {
		return errors.New("missing refund_ref")
	}

	var r Refund
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&r, "provider = ? AND provider_ref = ?", provider, ev.RefundRef).Error; err != nil {
		return err
	}
	if r.Status == StatusFailed {
		return nil
	}

	now := time.Now()
	if err := tx.WithContext(ctx).Model(&Refund{}).
		Where("id = ?", r.ID).
		Updates(map[string]any{
			"status":        StatusFailed,
			"error_message": "provider webhook: failed",
			"updated_at":    now,
		}).Error; err != nil {
		return err
	}

	// ledger: refund_failed (0)
	_ = ensureFinancialEntry(ctx, tx, orders.FinancialEntry{
		ID:          uuid.NewString(),
		OrderID:     r.OrderID,
		Event:       "refund_failed",
		AmountCents: 0,
		Currency:    r.Currency,
		RefType:     "refund",
		RefID:       r.ID,
		CreatedAt:   now,
	})
	return nil
}

func ensureFinancialEntry(ctx context.Context, tx *gorm.DB, e orders.FinancialEntry) error {
	var cnt int64
	if err := tx.WithContext(ctx).
		Model(&orders.FinancialEntry{}).
		Where("ref_type = ? AND ref_id = ? AND event = ?", e.RefType, e.RefID, e.Event).
		Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	return tx.WithContext(ctx).Create(&e).Error
}

func isDup(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
