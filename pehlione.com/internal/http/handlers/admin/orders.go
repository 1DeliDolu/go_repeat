package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/payments"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type OrdersHandler struct {
	DB        *gorm.DB
	RefundSvc *payments.RefundService
}

func NewOrdersHandler(db *gorm.DB, refundSvc *payments.RefundService) *OrdersHandler {
	return &OrdersHandler{DB: db, RefundSvc: refundSvc}
}

func (h *OrdersHandler) List(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	status := strings.TrimSpace(c.Query("status"))
	page := parseInt(c.Query("page"), 1)

	repo := orders.NewRepo(h.DB)
	res, err := repo.AdminList(c.Request.Context(), orders.AdminListParams{
		Q: q, Status: status, Page: page, PageSize: 30,
	})
	if err != nil {
		c.Error(apperr.Wrap(err))
		return
	}

	// view model (simple)
	items := make([]view.AdminOrderListItem, 0, len(res.Items))
	for _, o := range res.Items {
		items = append(items, view.AdminOrderListItem{
			ID:         o.ID,
			Status:     o.Status,
			Total:      view.MoneyFromCents(o.TotalCents, o.Currency),
			CreatedAt:  o.CreatedAt.Format("2006-01-02 15:04"),
			UserID:     ptrStr(o.UserID),
			GuestEmail: ptrStr(o.GuestEmail),
		})
	}

	totalPages := pagesFromTotal(res.Total, 30)
	render.Component(c, http.StatusOK, pages.AdminOrdersList(
		middleware.GetFlash(c),
		view.AdminOrdersListPage{
			Items:      items,
			Q:          q,
			Status:     status,
			Page:       page,
			TotalPages: totalPages,
		},
	))
}

func (h *OrdersHandler) Detail(c *gin.Context) {
	id := c.Param("id")

	repo := orders.NewRepo(h.DB)
	o, items, ev, err := repo.AdminGetDetail(c.Request.Context(), id)
	if err != nil {
		c.Error(apperr.NotFoundErr("Order bulunamadı."))
		return
	}

	vm := view.AdminOrderDetail{
		ID:         o.ID,
		Status:     o.Status,
		Currency:   o.Currency,
		Subtotal:   view.MoneyFromCents(o.SubtotalCents, o.Currency),
		Shipping:   view.MoneyFromCents(o.ShippingCents, o.Currency),
		Tax:        view.MoneyFromCents(o.TaxCents, o.Currency),
		Discount:   view.MoneyFromCents(o.DiscountCents, o.Currency),
		Total:      view.MoneyFromCents(o.TotalCents, o.Currency),
		UserID:     ptrStr(o.UserID),
		GuestEmail: ptrStr(o.GuestEmail),
		CreatedAt:  o.CreatedAt.Format("2006-01-02 15:04"),
	}

	for _, it := range items {
		vm.Items = append(vm.Items, view.AdminOrderItem{
			ProductName: it.ProductName,
			SKU:         it.SKU,
			Options:     string(it.OptionsJSON),
			Qty:         it.Quantity,
			Unit:        view.MoneyFromCents(it.UnitPriceCents, it.Currency),
			Line:        view.MoneyFromCents(it.LineTotalCents, it.Currency),
		})
	}
	for _, e := range ev {
		vm.Events = append(vm.Events, view.AdminOrderEvent{
			Action:      e.Action,
			From:        e.FromStatus,
			To:          e.ToStatus,
			ActorUserID: e.ActorUserID,
			Note:        ptrStr(e.Note),
			At:          e.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	// Financial entries
	fin, _ := repo.AdminListFinancial(c.Request.Context(), id)
	for _, f := range fin {
		sign := "+"
		if f.AmountCents < 0 {
			sign = ""
		}
		vm.Financial = append(vm.Financial, view.AdminOrderFinancialEntry{
			Event:       f.Event,
			AmountCents: f.AmountCents,
			AmountStr:   sign + view.MoneyFromCents(f.AmountCents, f.Currency),
			Currency:    f.Currency,
			RefType:     f.RefType,
			RefID:       f.RefID,
			At:          f.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	render.Component(c, http.StatusOK, pages.AdminOrderDetail(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		vm,
	))
}

func (h *OrdersHandler) Action(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action") // ship|deliver|cancel|refund

	u, ok := middleware.CurrentUser(c)
	if !ok {
		c.Error(apperr.ForbiddenErr("Giriş gerekli."))
		return
	}

	note := strings.TrimSpace(c.PostForm("note"))
	confirm := c.PostForm("confirm") == "1"
	if !confirm {
		c.Redirect(http.StatusFound, "/admin/orders/"+id)
		return
	}

	// Handle refund separately via RefundService
	if action == "refund" {
		idem := strings.TrimSpace(c.PostForm("idempotency_key"))
		if idem == "" {
			idem = randHex(16)
		}

		amountCents := parseInt(c.PostForm("amount_cents"), 0)

		res, err := h.RefundSvc.RefundOrder(c.Request.Context(), payments.RefundOrderInput{
			OrderID:        id,
			ActorUserID:    u.ID,
			IdempotencyKey: idem,
			AmountCents:    amountCents,
			Reason:         note,
		})
		if err != nil {
			if errors.Is(err, payments.ErrNotRefundable) {
				c.Error(apperr.InvalidErr("Order not refundable.", nil))
				return
			}
			c.Error(apperr.Wrap(err))
			return
		}

		msg := "Refund processed: " + res.Status
		if res.Idempotent {
			msg += " (idempotent)"
		}
		c.Redirect(http.StatusFound, "/admin/orders/"+id)
		return
	}

	// Handle other actions via state machine
	svc := orders.NewAdminService(h.DB)
	err := svc.Transition(c.Request.Context(), orders.TransitionInput{
		OrderID:     id,
		ActorUserID: u.ID,
		Action:      action,
		Note:        note,
	})
	if err != nil {
		if errors.Is(err, orders.ErrInvalidTransition) {
			c.Error(apperr.Wrap(apperr.InvalidErr("Geçersiz status geçişi.", nil)))
			return
		}
		c.Error(apperr.Wrap(err))
		return
	}

	c.Redirect(http.StatusFound, "/admin/orders/"+id)
}

func parseInt(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 {
		return def
	}
	return n
}

func pagesFromTotal(total int64, size int) int {
	if size <= 0 || total <= 0 {
		return 1
	}
	p := int((total + int64(size) - 1) / int64(size))
	if p < 1 {
		return 1
	}
	return p
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func randHex(nBytes int) string {
	return uuid.New().String()[:nBytes]
}
