package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/payments"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type OrdersHandler struct {
	DB     *gorm.DB
	Flash  *flash.Codec
	PaySvc *payments.Service
}

func NewOrdersHandler(db *gorm.DB, fl *flash.Codec, pay *payments.Service) *OrdersHandler {
	return &OrdersHandler{DB: db, Flash: fl, PaySvc: pay}
}

func (h *OrdersHandler) Detail(c *gin.Context) {
	id := c.Param("id")

	o, items, err := orders.NewRepo(h.DB).GetWithItems(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, apperr.NotFoundErr("Sipariş bulunamadı."))
		return
	}

	vm := view.OrderDetail{
		ID:       o.ID,
		Status:   o.Status,
		Currency: o.Currency,
		Subtotal: view.MoneyFromCents(o.SubtotalCents, o.Currency),
		Shipping: view.MoneyFromCents(o.ShippingCents, o.Currency),
		Tax:      view.MoneyFromCents(o.TaxCents, o.Currency),
		Discount: view.MoneyFromCents(o.DiscountCents, o.Currency),
		Total:    view.MoneyFromCents(o.TotalCents, o.Currency),
	}

	for _, it := range items {
		vm.Items = append(vm.Items, view.OrderItem{
			ProductName: it.ProductName,
			SKU:         it.SKU,
			Options:     string(it.OptionsJSON),
			Qty:         it.Quantity,
			PriceEach:   view.MoneyFromCents(it.UnitPriceCents, it.Currency),
			LineTotal:   view.MoneyFromCents(it.LineTotalCents, it.Currency),
		})
	}

	render.Component(c, http.StatusOK, pages.OrderDetail(
		middleware.GetFlash(c),
		vm,
	))
}

func (h *OrdersHandler) PayGet(c *gin.Context) {
	id := c.Param("id")

	o, _, err := orders.NewRepo(h.DB).GetWithItems(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, apperr.NotFoundErr("Sipariş bulunamadı."))
		return
	}

	// Eğer user order ise auth zorunlu
	if o.UserID != nil {
		u, ok := middleware.CurrentUser(c)
		if !ok || u.ID != *o.UserID {
			middleware.Fail(c, apperr.ForbiddenErr("Erişim yok."))
			return
		}
	}

	if o.Status != "created" {
		render.RedirectWithFlash(c, h.Flash, "/orders/"+o.ID, view.FlashWarning, "Sipariş ödeme için uygun değil.")
		return
	}

	idem := randHex(16)
	render.Component(c, http.StatusOK, pages.OrderPay(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		o.ID,
		view.MoneyFromCents(o.TotalCents, o.Currency),
		idem,
	))
}

func (h *OrdersHandler) PayPost(c *gin.Context) {
	id := c.Param("id")
	idem := c.PostForm("idempotency_key")

	o, _, err := orders.NewRepo(h.DB).GetWithItems(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, apperr.NotFoundErr("Sipariş bulunamadı."))
		return
	}

	var actor *string
	if o.UserID != nil {
		u, ok := middleware.CurrentUser(c)
		if !ok || u.ID != *o.UserID {
			middleware.Fail(c, apperr.ForbiddenErr("Erişim yok."))
			return
		}
		actor = &u.ID
	}

	res, err := h.PaySvc.PayOrder(c.Request.Context(), payments.PayOrderInput{
		OrderID:        o.ID,
		ActorUserID:    actor,
		IdempotencyKey: idem,
		ReturnURL:      "/orders/" + o.ID,
		CancelURL:      "/orders/" + o.ID,
	})
	if err != nil {
		if errors.Is(err, payments.ErrOrderNotPayable) {
			render.RedirectWithFlash(c, h.Flash, "/orders/"+o.ID, view.FlashWarning, "Sipariş ödeme için uygun değil.")
			return
		}
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	if res.Status == payments.StatusSucceeded {
		render.RedirectWithFlash(c, h.Flash, "/orders/"+o.ID, view.FlashSuccess, "Ödeme başarılı. Sipariş ödendi.")
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/orders/"+o.ID, view.FlashError, "Ödeme başarısız.")
}
