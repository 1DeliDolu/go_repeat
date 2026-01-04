package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type AccountOrdersHandler struct {
	ordersRepo *orders.Repo
}

func NewAccountOrdersHandler(ordersRepo *orders.Repo) *AccountOrdersHandler {
	return &AccountOrdersHandler{ordersRepo: ordersRepo}
}

func (h *AccountOrdersHandler) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	status := c.Query("status")
	const pageSize = 20

	result, err := h.ordersRepo.ListByUser(c, orders.ListByUserParams{
		UserID:   user.ID,
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		c.Error(apperr.Wrap(err))
		return
	}

	items := make([]view.AccountOrderListItem, len(result.Items))
	for i, item := range result.Items {
		items[i] = view.AccountOrderListItem{
			ID:         item.Order.ID,
			Number:     item.Order.ID[:8], // Use short ID as order number
			CreatedAt:  item.Order.CreatedAt,
			Status:     item.Order.Status,
			TotalCents: item.Order.TotalCents,
			Currency:   item.Order.Currency,
			ItemCount:  item.Count,
			PaidAt:     item.Order.PaidAt,
		}
	}

	pagesTotal := int((result.Total + int64(pageSize) - 1) / int64(pageSize))
	if pagesTotal < 1 {
		pagesTotal = 1
	}

	pageView := view.AccountOrdersPage{
		Items:          items,
		Total:          result.Total,
		Page:           page,
		PageSize:       pageSize,
		FilterStatus:   status,
		Statuses:       []string{"created", "processing", "shipped", "delivered", "cancelled"},
		IsPreviousPage: page > 1,
		IsNextPage:     page < pagesTotal,
	}

	render.Component(c, http.StatusOK, pages.AccountOrders(
		middleware.GetFlash(c),
		pageView,
	))
}
