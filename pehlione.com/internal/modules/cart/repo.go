package cart

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) GetOrCreateUserCart(ctx context.Context, userID string) (Cart, error) {
	if userID == "" {
		return Cart{}, errors.New("userID cannot be empty")
	}

	var c Cart
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "open").
		Attrs(Cart{
			ID:     uuid.NewString(),
			UserID: &userID, // CRITICAL: Set user_id when creating new cart
			Status: "open",
		}).
		FirstOrCreate(&c).Error
	if err != nil {
		return Cart{}, err
	}

	// Ensure cart belongs to user and is marked open
	if c.UserID == nil || *c.UserID != userID {
		c.UserID = &userID
		if err := r.db.WithContext(ctx).Model(&c).Update("user_id", userID).Error; err != nil {
			return Cart{}, err
		}
	}
	if c.Status != "open" {
		c.Status = "open"
		if err := r.db.WithContext(ctx).Model(&c).Update("status", "open").Error; err != nil {
			return Cart{}, err
		}
	}
	log.Printf("GetOrCreateUserCart: user=%s, cart=%s (status=%s)", userID, c.ID, c.Status)
	return c, nil
}

func (r *Repo) GetCart(ctx context.Context, cartID string) (Cart, error) {
	var c Cart
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Items.Variant").
		First(&c, "id = ?", cartID).Error
	return c, err
}

func (r *Repo) AddItem(ctx context.Context, cartID string, variantID string, qty int) error {
	// Check if item already exists
	var existing CartItem
	err := r.db.WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		First(&existing).Error

	if err == nil {
		// Item exists, increment quantity
		newQty := existing.Quantity + qty
		if err := r.db.WithContext(ctx).
			Model(&CartItem{}).
			Where("cart_id = ? AND variant_id = ?", cartID, variantID).
			Update("quantity", newQty).Error; err != nil {
			return err
		}
		return r.touchCart(ctx, cartID)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // Real error, not just missing record
	}

	// Item doesn't exist, create new
	item := CartItem{
		ID:        uuid.NewString(),
		CartID:    cartID,
		VariantID: variantID,
		Quantity:  qty,
	}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		return err
	}
	return r.touchCart(ctx, cartID)
}

func (r *Repo) UpdateItemQty(ctx context.Context, cartID string, variantID string, qty int) error {
	if qty <= 0 {
		if err := r.db.WithContext(ctx).Where("cart_id = ? AND variant_id = ?", cartID, variantID).Delete(&CartItem{}).Error; err != nil {
			return err
		}
		return r.touchCart(ctx, cartID)
	}
	if err := r.db.WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		Update("quantity", qty).Error; err != nil {
		return err
	}
	return r.touchCart(ctx, cartID)
}

func (r *Repo) RemoveItem(ctx context.Context, cartID string, variantID string) error {
	if err := r.db.WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		Delete(&CartItem{}).Error; err != nil {
		return err
	}
	return r.touchCart(ctx, cartID)
}

func (r *Repo) ClearCart(ctx context.Context, cartID string) error {
	return r.db.WithContext(ctx).Where("cart_id = ?", cartID).Delete(&CartItem{}).Error
}

func (r *Repo) touchCart(ctx context.Context, cartID string) error {
	return r.db.WithContext(ctx).
		Model(&Cart{}).
		Where("id = ?", cartID).
		UpdateColumn("updated_at", gorm.Expr("CURRENT_TIMESTAMP(3)")).Error
}
