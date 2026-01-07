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
	log.Printf("ListByUser: fetching orders for user_id=%s", in.UserID)
	page := in.Page
	if page < 1 {
		page = 1
	}
	size := in.PageSize
	if size < 1 || size > 100 {
		size = 20
	}
	status := strings.TrimSpace(in.Status)

	// Get user email to also include guest orders
	var userEmail string
	if err := r.db.WithContext(ctx).Table("users").Select("email").Where("id = ?", in.UserID).Scan(&userEmail).Error; err != nil {
		log.Printf("ListByUser: failed to get user email: %v", err)
		userEmail = ""
	}

	// Include both user_id orders AND guest orders with matching email
	q := r.db.WithContext(ctx).Model(&Order{}).
		Where("user_id = ? OR (user_id IS NULL AND guest_email = ?)", in.UserID, userEmail)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		log.Printf("ListByUser: count error: %v", err)
		return ListByUserResult{}, err
	}
	log.Printf("ListByUser: found %d total orders for user %s (email=%s)", total, in.UserID, userEmail)

	var orders []Order
	if err := q.
		Order("created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&orders).Error; err != nil {
		log.Printf("ListByUser: find error: %v", err)
		return ListByUserResult{}, err
	}
	log.Printf("ListByUser: fetched %d orders on page %d", len(orders), page)

	items := make([]ListByUserItem, len(orders))
	for i, o := range orders {
		var count int64
		if err := r.db.WithContext(ctx).Model(&OrderItem{}).Where("order_id = ?", o.ID).Count(&count).Error; err != nil {
			count = 0
		}
		log.Printf("ListByUser: order %s has %d items", o.ID, count)
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
