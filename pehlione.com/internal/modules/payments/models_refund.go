package payments

import "time"

type Refund struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	OrderID   string `gorm:"type:char(36);not null;index:ix_refunds_order_id"`
	PaymentID string `gorm:"type:char(36);not null;index:ix_refunds_payment_id"`

	Provider    string  `gorm:"type:varchar(64);not null"`
	ProviderRef *string `gorm:"type:varchar(128)"`

	Status         string `gorm:"type:varchar(32);not null"`
	AmountCents    int    `gorm:"not null"`
	Currency       string `gorm:"type:char(3);not null"`
	IdempotencyKey string `gorm:"type:varchar(64);not null"`

	Reason       *string `gorm:"type:varchar(255)"`
	ErrorMessage *string `gorm:"type:varchar(255)"`

	CreatedAt time.Time `gorm:"type:datetime(3);not null"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null"`
}

func (Refund) TableName() string { return "refunds" }
