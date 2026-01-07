package shipping

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) ListByOrder(ctx context.Context, orderID string) ([]Shipment, error) {
	// Normalize UUID to lowercase
	orderID = strings.ToLower(strings.TrimSpace(orderID))

	var shipments []Shipment
	err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&shipments, "order_id = ?", orderID).Error
	return shipments, err
}
