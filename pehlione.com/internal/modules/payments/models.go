package payments

import "time"

const (
	StatusInitiated = "initiated"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

type Payment struct {
	ID             string    `gorm:"type:char(36);primaryKey"`
	OrderID        string    `gorm:"type:char(36);not null;index:ix_payments_order_id"`
	Provider       string    `gorm:"type:varchar(64);not null"`
	ProviderRef    *string   `gorm:"type:varchar(128)"`
	Status         string    `gorm:"type:varchar(32);not null"`
	AmountCents    int       `gorm:"not null"`
	Currency       string    `gorm:"type:char(3);not null"`
	IdempotencyKey string    `gorm:"type:varchar(64);not null"`
	ErrorMessage   *string   `gorm:"type:varchar(255)"`
	CreatedAt      time.Time `gorm:"type:datetime(3);not null"`
	UpdatedAt      time.Time `gorm:"type:datetime(3);not null"`
}

func (Payment) TableName() string { return "payments" }
