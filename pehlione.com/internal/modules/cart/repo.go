package cart

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) GetOrCreateUserCart(ctx context.Context, userID string) (Cart, error) {
	var c Cart
	err := r.db.WithContext(ctx).FirstOrCreate(&c, Cart{UserID: &userID}).Error
	return c, err
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
