package orders

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

type ListByUserParams struct {
	UserID   string
	Page     int
	PageSize int
	Status   string // optional filter
}

type ListByUserResult struct {
	Items []ListByUserItem
	Total int64
}

type ListByUserItem struct {
	Order Order
	Count int
}

func (r *Repo) ListByUser(ctx context.Context, in ListByUserParams) (ListByUserResult, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	size := in.PageSize
	if size < 1 || size > 100 {
		size = 20
	}
	status := strings.TrimSpace(in.Status)

	q := r.db.WithContext(ctx).Model(&Order{}).Where("user_id = ?", in.UserID)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ListByUserResult{}, err
	}

	var orders []Order
	if err := q.
		Order("created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&orders).Error; err != nil {
		return ListByUserResult{}, err
	}

	items := make([]ListByUserItem, len(orders))
	for i, o := range orders {
		var count int64
		if err := r.db.WithContext(ctx).Model(&OrderItem{}).Where("order_id = ?", o.ID).Count(&count).Error; err != nil {
			count = 0
		}
		items[i] = ListByUserItem{Order: o, Count: int(count)}
	}

	return ListByUserResult{Items: items, Total: total}, nil
}

func (r *Repo) GetWithItems(ctx context.Context, id string) (Order, []OrderItem, error) {
	var o Order
	if err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error; err != nil {
		return Order{}, nil, err
	}
	var items []OrderItem
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&items, "order_id = ?", id).Error; err != nil {
		return Order{}, nil, err
	}
	return o, items, nil
}
