package view

import (
	"time"
)

type AccountOrderListItem struct {
	ID         string
	Number     string
	CreatedAt  time.Time
	Status     string
	TotalCents int64
	Currency   string
	ItemCount  int
	PaidAt     *time.Time
}

type AccountInfo struct {
	Email     string
	Status    string
	CreatedAt time.Time
	Verified  bool
}

type ChangePasswordForm struct {
	Current string
	New     string
	Confirm string
}

type AccountOrdersPage struct {
	Account        AccountInfo
	Items          []AccountOrderListItem
	Total          int64
	Page           int
	PageSize       int
	FilterStatus   string
	Statuses       []string
	IsPreviousPage bool
	IsNextPage     bool
	CSRFToken      string
	PasswordErrors map[string]string
}

func (p AccountOrdersPage) PagesTotal() int {
	if p.Total == 0 {
		return 1
	}
	return int((p.Total + int64(p.PageSize) - 1) / int64(p.PageSize))
}
