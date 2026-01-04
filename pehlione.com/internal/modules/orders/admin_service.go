package orders

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidTransition = errors.New("invalid order status transition")
	ErrNotActionable     = errors.New("order not actionable")
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService { return &AdminService{db: db} }

type TransitionInput struct {
	OrderID     string
	ActorUserID string // admin user id
	Action      string // ship|deliver|cancel|refund
	Note        string
}

func (s *AdminService) Transition(ctx context.Context, in TransitionInput) error {
	if in.OrderID == "" || in.ActorUserID == "" || in.Action == "" {
		return ErrNotActionable
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var o Order

		// row lock
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&o, "id = ?", in.OrderID).Error; err != nil {
			return err
		}

		from := o.Status
		to, err := nextStatus(from, in.Action)
		if err != nil {
			return err
		}
		if from == to {
			return ErrInvalidTransition
		}

		now := time.Now()
		updates := map[string]any{
			"status":     to,
			"updated_at": now,
		}
		if to == "paid" && o.PaidAt == nil {
			updates["paid_at"] = now
		}

		if err := tx.WithContext(ctx).
			Model(&Order{}).
			Where("id = ? AND status = ?", o.ID, from). // optimistic guard
			Updates(updates).Error; err != nil {
			return err
		}

		var notePtr *string
		if n := stringsTrim(in.Note); n != "" {
			notePtr = &n
		}

		ev := OrderEvent{
			ID:          uuid.NewString(),
			OrderID:     o.ID,
			ActorUserID: in.ActorUserID,
			Action:      in.Action,
			FromStatus:  from,
			ToStatus:    to,
			Note:        notePtr,
			CreatedAt:   now,
		}
		return tx.WithContext(ctx).Create(&ev).Error
	})
}

func nextStatus(from, action string) (string, error) {
	switch action {
	case "cancel":
		if from == "created" {
			return "cancelled", nil
		}
		return "", ErrInvalidTransition
	case "ship":
		if from == "paid" {
			return "shipped", nil
		}
		return "", ErrInvalidTransition
	case "deliver":
		if from == "shipped" {
			return "delivered", nil
		}
		return "", ErrInvalidTransition
	case "refund":
		if from == "paid" {
			return "refunded", nil
		}
		return "", ErrInvalidTransition
	default:
		return "", ErrInvalidTransition
	}
}

func stringsTrim(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
