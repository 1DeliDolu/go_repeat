package cart

import "time"

type Cart struct {
	ID        string  `gorm:"type:char(36);primaryKey"`
	UserID    *string `gorm:"type:char(36);index:ix_carts_user_id"`
	Items     []CartItem
	CreatedAt time.Time `gorm:"type:datetime(3);not null"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null"`
}

func (Cart) TableName() string { return "carts" }

type CartItem struct {
	ID        string  `gorm:"type:char(36);primaryKey"`
	CartID    string  `gorm:"type:char(36);not null;index:ix_cart_items_cart_id"`
	VariantID string  `gorm:"type:char(36);not null;index:ix_cart_items_variant_id"`
	Quantity  int     `gorm:"not null"`
	Variant   Variant `gorm:"foreignKey:VariantID;references:ID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CartItem) TableName() string { return "cart_items" }

// Variant (snapshot from product_variants)
type Variant struct {
	ID         string
	ProductID  string
	SKU        string
	PriceCents int
	Currency   string
	Stock      int
	Options    []byte
}
