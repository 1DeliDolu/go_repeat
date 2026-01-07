package orders

import (
	"context"
	"log"
	"strings"
)

type AdminListParams struct {
	Q        string
	Status   string
	Page     int
	PageSize int
}

type AdminListResult struct {
	Items []Order
	Total int64
}

func (r *Repo) AdminList(ctx context.Context, in AdminListParams) (AdminListResult, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	size := in.PageSize
	if size < 1 || size > 100 {
		size = 30
	}

	q := strings.TrimSpace(in.Q)
	status := strings.TrimSpace(in.Status)

	base := r.db.WithContext(ctx).Model(&Order{})
	if status != "" {
		base = base.Where("status = ?", status)
	}
	if q != "" {
		like := "%" + q + "%"
		// order id or guest email (simple)
		base = base.Where("(id LIKE ? OR guest_email LIKE ?)", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return AdminListResult{}, err
	}

	var items []Order
	if err := base.
		Order("created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&items).Error; err != nil {
		return AdminListResult{}, err
	}

	return AdminListResult{Items: items, Total: total}, nil
}

func (r *Repo) AdminGetDetail(ctx context.Context, orderID string) (Order, []OrderItem, []OrderEvent, error) {
	// Normalize UUID to lowercase
	orderID = strings.ToLower(strings.TrimSpace(orderID))
	log.Printf("[orders.AdminGetDetail] Starting: id=%s", orderID)

	o, items, err := r.GetWithItems(ctx, orderID)
	if err != nil {
		log.Printf("[orders.AdminGetDetail] GetWithItems failed: id=%s, err=%v", orderID, err)
		return Order{}, nil, nil, err
	}
	log.Printf("[orders.AdminGetDetail] GetWithItems succeeded: id=%s", orderID)

	var ev []OrderEvent
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&ev, "order_id = ?", orderID).Error; err != nil {
		log.Printf("[orders.AdminGetDetail] Events query failed: id=%s, err=%v", orderID, err)
		return Order{}, nil, nil, err
	}
	log.Printf("[orders.AdminGetDetail] Events found: id=%s, count=%d", orderID, len(ev))
	return o, items, ev, nil
}

func (r *Repo) AdminListFinancial(ctx context.Context, orderID string) ([]FinancialEntry, error) {
	// Normalize UUID to lowercase
	orderID = strings.ToLower(strings.TrimSpace(orderID))

	var out []FinancialEntry
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&out, "order_id = ?", orderID).Error
	return out, err
}
