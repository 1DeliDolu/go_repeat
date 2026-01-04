package payments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pehlione.com/app/internal/modules/orders"
)

type Service struct {
	db       *gorm.DB
	provider Provider
}

func NewService(db *gorm.DB, p Provider) *Service {
	return &Service{db: db, provider: p}
}

type PayOrderInput struct {
	OrderID        string
	ActorUserID    *string // order.user_id varsa match etmeli
	IdempotencyKey string  // SSR hidden field
	ReturnURL      string
	CancelURL      string
}

type PayOrderResult struct {
	OrderID    string
	PaymentID  string
	Status     string
	Idempotent bool
}

func (s *Service) PayOrder(ctx context.Context, in PayOrderInput) (PayOrderResult, error) {
	if in.OrderID == "" || in.IdempotencyKey == "" {
		return PayOrderResult{}, ErrOrderNotPayable
	}

	// Phase-1: order lock + idempotency check + payment initiated create
	var createdPayment Payment
	var ord orders.Order

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Order row lock
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ord, "id = ?", in.OrderID).Error; err != nil {
			return err
		}

		// AuthZ: order user'ı varsa actor match
		if ord.UserID != nil && in.ActorUserID != nil && *ord.UserID != *in.ActorUserID {
			return ErrForbidden
		}
		if ord.UserID != nil && in.ActorUserID == nil {
			return ErrForbidden
		}

		// Status gate
		if ord.Status != "created" {
			return ErrOrderNotPayable
		}

		// Idempotency: aynı order+key payment varsa döndür
		var existing Payment
		e := tx.WithContext(ctx).First(&existing, "order_id = ? AND idempotency_key = ?", ord.ID, in.IdempotencyKey).Error
		if e == nil {
			createdPayment = existing
			return nil
		}
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}

		now := time.Now()
		createdPayment = Payment{
			ID:             uuid.NewString(),
			OrderID:        ord.ID,
			Provider:       s.provider.Name(),
			ProviderRef:    nil,
			Status:         StatusInitiated,
			AmountCents:    ord.TotalCents,
			Currency:       ord.Currency,
			IdempotencyKey: in.IdempotencyKey,
			ErrorMessage:   nil,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		return tx.WithContext(ctx).Create(&createdPayment).Error
	})
	if err != nil {
		return PayOrderResult{}, err
	}

	// Eğer idempotency ile var olan payment geldi ve succeeded ise hemen dön
	if createdPayment.Status == StatusSucceeded {
		return PayOrderResult{OrderID: ord.ID, PaymentID: createdPayment.ID, Status: createdPayment.Status, Idempotent: true}, nil
	}

	// Phase-2: provider çağrısı (tx dışında)
	resp, perr := s.provider.CreatePayment(ctx, CreatePaymentRequest{
		OrderID:        ord.ID,
		AmountCents:    ord.TotalCents,
		Currency:       ord.Currency,
		IdempotencyKey: in.IdempotencyKey,
		ReturnURL:      in.ReturnURL,
		CancelURL:      in.CancelURL,
	})

	// Phase-3: payment finalize + order update (tx)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		updates := map[string]any{"updated_at": now}
		if resp.ProviderRef != "" {
			updates["provider_ref"] = resp.ProviderRef
		}

		if perr != nil {
			msg := perr.Error()
			updates["status"] = StatusFailed
			updates["error_message"] = msg

			if err := tx.WithContext(ctx).Model(&Payment{}).
				Where("id = ?", createdPayment.ID).
				Updates(updates).Error; err != nil {
				return err
			}
			return nil
		}

		// async: initiated (webhook will finalize)
		if resp.Status == StatusInitiated {
			updates["status"] = StatusInitiated
			if err := tx.WithContext(ctx).Model(&Payment{}).
				Where("id = ?", createdPayment.ID).
				Updates(updates).Error; err != nil {
				return err
			}
			// Don't touch order; webhook will finalize it
			return nil
		}

		// mock: succeeded
		if resp.Status == StatusSucceeded {
			updates["status"] = StatusSucceeded
			updates["error_message"] = nil

			if err := tx.WithContext(ctx).Model(&Payment{}).
				Where("id = ?", createdPayment.ID).
				Updates(updates).Error; err != nil {
				return err
			}

			// Financial ledger: payment_succeeded entry
			entry := orders.FinancialEntry{
				ID:          uuid.NewString(),
				OrderID:     ord.ID,
				Event:       "payment_succeeded",
				AmountCents: ord.TotalCents, // +in
				Currency:    ord.Currency,
				RefType:     "payment",
				RefID:       createdPayment.ID,
				CreatedAt:   now,
			}
			if err := tx.WithContext(ctx).Create(&entry).Error; err != nil {
				return err
			}

			// order status -> paid
			paidAt := now
			if err := tx.WithContext(ctx).Model(&orders.Order{}).
				Where("id = ? AND status = 'created'", ord.ID).
				Updates(map[string]any{
					"status":     "paid",
					"paid_at":    &paidAt,
					"updated_at": now,
				}).Error; err != nil {
				return err
			}
			return nil
		}

		// default: failed
		updates["status"] = StatusFailed
		if err := tx.WithContext(ctx).Model(&Payment{}).
			Where("id = ?", createdPayment.ID).
			Updates(updates).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return PayOrderResult{}, err
	}

	// reload minimal: status için
	finalStatus := resp.Status
	if perr != nil {
		finalStatus = StatusFailed
	}

	return PayOrderResult{
		OrderID:    ord.ID,
		PaymentID:  createdPayment.ID,
		Status:     finalStatus,
		Idempotent: false,
	}, nil
}
