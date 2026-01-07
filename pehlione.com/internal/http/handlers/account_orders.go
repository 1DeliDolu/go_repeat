package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/auth"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type AccountOrdersHandler struct {
	ordersRepo *orders.Repo
	authRepo   *auth.Repo
	Flash      *flash.Codec
}

func NewAccountOrdersHandler(ordersRepo *orders.Repo, authRepo *auth.Repo, flashCodec *flash.Codec) *AccountOrdersHandler {
	return &AccountOrdersHandler{ordersRepo: ordersRepo, authRepo: authRepo, Flash: flashCodec}
}

func (h *AccountOrdersHandler) buildPage(c *gin.Context, user auth.User, page, pageSize int, status string, passwordErrors map[string]string) (view.AccountOrdersPage, error) {
	offset := (page - 1) * pageSize

	result, err := h.ordersRepo.ListByUser(c.Request.Context(), orders.ListByUserParams{
		UserID:   user.ID,
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		return view.AccountOrdersPage{}, err
	}

	items := make([]view.AccountOrderListItem, len(result.Items))
	for i, item := range result.Items {
		orderNum := item.Order.ID
		if len(orderNum) > 8 {
			orderNum = orderNum[:8]
		}

		items[i] = view.AccountOrderListItem{
			ID:         item.Order.ID,
			Number:     orderNum,
			CreatedAt:  item.Order.CreatedAt,
			Status:     item.Order.Status,
			TotalCents: int64(item.Order.TotalCents),
			Currency:   item.Order.Currency,
			ItemCount:  item.Count,
			PaidAt:     item.Order.PaidAt,
		}
	}

	return view.AccountOrdersPage{
		Account: view.AccountInfo{
			Email:     user.Email,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			Verified:  user.EmailVerifiedAt != nil,
		},
		Items:          items,
		Total:          result.Total,
		Page:           page,
		PageSize:       pageSize,
		FilterStatus:   status,
		Statuses:       []string{"pending", "paid", "shipped", "delivered", "cancelled"},
		IsPreviousPage: page > 1,
		IsNextPage:     offset+pageSize < int(result.Total),
		CSRFToken:      middleware.GetCSRFToken(c),
		PasswordErrors: passwordErrors,
	}, nil
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

	fullUser, err := h.authRepo.GetByID(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(apperr.Wrap(err))
		return
	}

	pageView, err := h.buildPage(c, *fullUser, page, pageSize, status, nil)
	if err != nil {
		c.Error(apperr.Wrap(err))
		return
	}

	render.Component(c, http.StatusOK, pages.AccountOrders(
		middleware.GetFlash(c),
		pageView,
	))
}
