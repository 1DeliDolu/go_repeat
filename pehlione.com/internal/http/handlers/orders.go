package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/payments"
	"pehlione.com/app/internal/pdf"
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

func (h *OrdersHandler) InvoicePDF(c *gin.Context) {
	id := c.Param("id")
	repo := orders.NewRepo(h.DB)

	o, items, err := repo.GetWithItems(c.Request.Context(), id)
	if err != nil {
		middleware.Fail(c, apperr.NotFoundErr("Sipariş bulunamadı."))
		return
	}

	if o.UserID != nil {
		u, ok := middleware.CurrentUser(c)
		if !ok || (u.ID != *o.UserID && u.Role != "admin") {
			middleware.Fail(c, apperr.ForbiddenErr("Bu siparişe erişim yok."))
			return
		}
	}

	addr := parseOrderAddress(o.ShippingAddressJSON)
	data := pdf.InvoiceData{
		Order:          o,
		Items:          items,
		ShippingLines:  formatOrderAddressLines(addr),
		ShippingMethod: view.ShippingLabel(addr.ShippingMethod),
		PaymentMethod:  view.PaymentMethodLabel(addr.PaymentMethod),
	}

	bytes, err := pdf.GenerateInvoice(data)
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	filename := fmt.Sprintf("pehlione-order-%s.pdf", o.ID)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Writer.Write(bytes)
}

type orderAddress struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Address1       string `json:"address1"`
	Address2       string `json:"address2"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	Country        string `json:"country"`
	Phone          string `json:"phone"`
	ShippingMethod string `json:"shipping_method"`
	PaymentMethod  string `json:"payment_method"`
}

func parseOrderAddress(data []byte) orderAddress {
	var addr orderAddress
	if len(data) == 0 {
		return addr
	}
	_ = json.Unmarshal(data, &addr)
	return addr
}

func formatOrderAddressLines(addr orderAddress) []string {
	var lines []string
	full := strings.TrimSpace(strings.TrimSpace(addr.FirstName) + " " + strings.TrimSpace(addr.LastName))
	if full != "" {
		lines = append(lines, full)
	}
	if addr.Address1 != "" {
		lines = append(lines, addr.Address1)
	}
	if addr.Address2 != "" {
		lines = append(lines, addr.Address2)
	}
	loc := strings.TrimSpace(strings.TrimSpace(addr.PostalCode) + " " + addr.City)
	if loc != "" {
		lines = append(lines, loc)
	}
	if addr.Country != "" {
		lines = append(lines, strings.ToUpper(addr.Country))
	}
	if addr.Phone != "" {
		lines = append(lines, "Tel: "+addr.Phone)
	}
	return lines
}
