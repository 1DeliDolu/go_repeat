package orders

import (
	"context"
	"log"
	"strings"

	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// DB returns the underlying database connection for direct queries.
func (r *Repo) DB() *gorm.DB { return r.db }

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
	// Normalize UUID to lowercase (standard UUID format in most databases)
	id = strings.ToLower(strings.TrimSpace(id))
	log.Printf("[orders.GetWithItems] Fetching order: id=%s", id)

	var o Order
	if err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error; err != nil {
		log.Printf("[orders.GetWithItems] Order not found: id=%s, err=%v", id, err)
		return Order{}, nil, err
	}
	log.Printf("[orders.GetWithItems] Order found: id=%s, status=%s", o.ID, o.Status)

	var items []OrderItem
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&items, "order_id = ?", id).Error; err != nil {
		log.Printf("[orders.GetWithItems] Items query failed: id=%s, err=%v", id, err)
		return Order{}, nil, err
	}
	log.Printf("[orders.GetWithItems] Items found: id=%s, count=%d", id, len(items))
	return o, items, nil
}
