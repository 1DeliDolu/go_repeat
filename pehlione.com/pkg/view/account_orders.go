package view

import (
	"time"
)

type AccountOrderListItem struct {
	ID         string
	Number     string
	CreatedAt  time.Time
	Status     string
	TotalCents int
	Currency   string
	ItemCount  int
	PaidAt     *time.Time
}

type AccountOrdersPage struct {
	Items          []AccountOrderListItem
	Total          int64
	Page           int
	PageSize       int
	FilterStatus   string
	Statuses       []string
	IsPreviousPage bool
	IsNextPage     bool
}

func (p AccountOrdersPage) PagesTotal() int {
	if p.Total == 0 {
		return 1
	}
	return int((p.Total + int64(p.PageSize) - 1) / int64(p.PageSize))
}
