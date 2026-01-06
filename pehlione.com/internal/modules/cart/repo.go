package cart

import (
	"context"
	"errors"

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
			Status: "open",
		}).
		FirstOrCreate(&c).Error
	if err != nil {
		return Cart{}, err
	}

	// Ensure cart belongs to user and is marked open
	c.UserID = &userID
	if c.Status != "open" {
		c.Status = "open"
		if err := r.db.WithContext(ctx).Model(&c).Update("status", "open").Error; err != nil {
			return Cart{}, err
		}
	}
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
		return r.db.WithContext(ctx).
			Model(&CartItem{}).
			Where("cart_id = ? AND variant_id = ?", cartID, variantID).
			Update("quantity", newQty).Error
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
	return r.db.WithContext(ctx).Create(&item).Error
}

func (r *Repo) UpdateItemQty(ctx context.Context, cartID string, variantID string, qty int) error {
	if qty <= 0 {
		return r.db.WithContext(ctx).Where("cart_id = ? AND variant_id = ?", cartID, variantID).Delete(&CartItem{}).Error
	}
	return r.db.WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		Update("quantity", qty).Error
}

func (r *Repo) RemoveItem(ctx context.Context, cartID string, variantID string) error {
	return r.db.WithContext(ctx).
		Where("cart_id = ? AND variant_id = ?", cartID, variantID).
		Delete(&CartItem{}).Error
}

func (r *Repo) ClearCart(ctx context.Context, cartID string) error {
	return r.db.WithContext(ctx).Where("cart_id = ?", cartID).Delete(&CartItem{}).Error
}
