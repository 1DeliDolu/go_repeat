package orders

import (
	"time"

	"gorm.io/datatypes"
)

type Order struct {
	ID string `gorm:"type:char(36);primaryKey"`

	UserID     *string `gorm:"type:char(36);index:ix_orders_user_id_created_at,priority:1"`
	GuestEmail *string `gorm:"type:varchar(255)"`

	Status   string `gorm:"type:varchar(32);not null"`
	Currency string `gorm:"type:char(3);not null"`

	SubtotalCents int `gorm:"not null"`
	TaxCents      int `gorm:"not null"`
	ShippingCents int `gorm:"not null"`
	DiscountCents int `gorm:"not null"`
	TotalCents    int `gorm:"not null"`

	ShippingAddressJSON datatypes.JSON `gorm:"type:json"`
	BillingAddressJSON  datatypes.JSON `gorm:"type:json"`

	IdempotencyKey *string    `gorm:"type:varchar(64);index"`
	PaidAt         *time.Time `gorm:"type:datetime(3)"`
	RefundedCents  int        `gorm:"not null;default:0"`
	RefundedAt     *time.Time `gorm:"type:datetime(3)"`

	CreatedAt time.Time `gorm:"type:datetime(3);not null"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null"`
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	OrderID   string `gorm:"type:char(36);not null;index:ix_order_items_order_id"`
	VariantID string `gorm:"type:char(36);not null;index:ix_order_items_variant_id"`

	ProductName string         `gorm:"type:varchar(255);not null"`
	SKU         string         `gorm:"type:varchar(64);not null"`
	OptionsJSON datatypes.JSON `gorm:"type:json;not null"`

	UnitPriceCents int    `gorm:"not null"`
	Currency       string `gorm:"type:char(3);not null"`
	Quantity       int    `gorm:"not null"`
	LineTotalCents int    `gorm:"not null"`

	CreatedAt time.Time `gorm:"type:datetime(3);not null"`
}

func (OrderItem) TableName() string { return "order_items" }

type OrderEvent struct {
	ID          string `gorm:"type:char(36);primaryKey"`
	OrderID     string `gorm:"type:char(36);not null;index:ix_order_events_order_id_created_at,priority:1"`
	ActorUserID string `gorm:"type:char(36);not null;index:ix_order_events_actor_created_at,priority:1"`

	Action     string  `gorm:"type:varchar(32);not null"`
	FromStatus string  `gorm:"type:varchar(32);not null"`
	ToStatus   string  `gorm:"type:varchar(32);not null"`
	Note       *string `gorm:"type:varchar(255)"`

	CreatedAt time.Time `gorm:"type:datetime(3);not null"`
}

func (OrderEvent) TableName() string { return "order_events" }

type FinancialEntry struct {
	ID          string    `gorm:"type:char(36);primaryKey"`
	OrderID     string    `gorm:"type:char(36);not null;index:ix_order_fin_entries_order_created,priority:1"`
	Event       string    `gorm:"type:varchar(32);not null"`
	AmountCents int       `gorm:"not null"` // +in, -out
	Currency    string    `gorm:"type:char(3);not null"`
	RefType     string    `gorm:"type:varchar(16);not null"`
	RefID       string    `gorm:"type:char(36);not null;index:ix_order_fin_entries_ref,priority:2"`
	CreatedAt   time.Time `gorm:"type:datetime(3);not null"`
}

func (FinancialEntry) TableName() string { return "order_financial_entries" }
