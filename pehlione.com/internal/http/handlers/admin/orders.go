package admin

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/payments"
	"pehlione.com/app/internal/modules/shipping"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type OrdersHandler struct {
	DB          *gorm.DB
	Flash       *flash.Codec
	RefundSvc   *payments.RefundService
	ShippingSvc *shipping.Service
}

func NewOrdersHandler(db *gorm.DB, fl *flash.Codec, refundSvc *payments.RefundService, shipSvc *shipping.Service) *OrdersHandler {
	return &OrdersHandler{DB: db, Flash: fl, RefundSvc: refundSvc, ShippingSvc: shipSvc}
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
	log.Printf("[admin/orders] Detail request: id=%s", id)

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		log.Printf("[admin/orders] Invalid UUID format: %v", err)
		c.Error(apperr.NotFoundErr("Order bulunamadı."))
		return
	}

	repo := orders.NewRepo(h.DB)

	// Debug: check if order exists in DB
	var testOrder orders.Order
	dbResult := h.DB.Where("id = ?", strings.ToLower(strings.TrimSpace(id))).First(&testOrder)
	log.Printf("[admin/orders] Direct DB query: id=%s, found=%v, err=%v",
		strings.ToLower(strings.TrimSpace(id)),
		dbResult.RowsAffected > 0,
		dbResult.Error)

	o, items, ev, err := repo.AdminGetDetail(c.Request.Context(), id)
	if err != nil {
		log.Printf("[admin/orders] AdminGetDetail error: %v", err)
		// Log the actual error for debugging
		c.Request.Header.Set("X-Debug-Error", err.Error())
		c.Error(apperr.NotFoundErr("Order bulunamadı."))
		return
	}
	log.Printf("[admin/orders] Successfully loaded order: id=%s", o.ID)

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

	shipRepo := shipping.NewRepo(h.DB)
	if shipments, err := shipRepo.ListByOrder(c.Request.Context(), id); err == nil {
		for _, s := range shipments {
			vm.Shipments = append(vm.Shipments, view.AdminShipment{
				ID:             s.ID,
				Carrier:        s.Carrier,
				Status:         s.Status,
				TrackingNumber: ptrStr(s.TrackingNumber),
				TrackingURL:    ptrStr(s.TrackingURL),
				LabelURL:       ptrStr(s.LabelURL),
				Note:           ptrStr(s.Note),
				ShippedAt:      formatTimePtr(s.ShippedAt),
				DeliveredAt:    formatTimePtr(s.DeliveredAt),
				Error:          ptrStr(s.ErrorMessage),
			})
		}
	}
	vm.ShippingAvailable = h.ShippingSvc != nil

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

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func randHex(nBytes int) string {
	return uuid.New().String()[:nBytes]
}

func (h *OrdersHandler) RefundForm(c *gin.Context) {
	id := c.Param("id")
	vm := pages.AdminOrderRefundVM{
		OrderID:   id,
		CSRFToken: middleware.GetCSRFToken(c),
		Error:     "",
	}
	render.Component(c, http.StatusOK, pages.AdminOrderRefund(vm))
}

func (h *OrdersHandler) Refund(c *gin.Context) {
	id := c.Param("id")

	u, ok := middleware.CurrentUser(c)
	if !ok {
		c.Error(apperr.ForbiddenErr("Giri�Y gerekli."))
		return
	}

	note := strings.TrimSpace(c.PostForm("note"))
	idem := randHex(16)

	res, err := h.RefundSvc.RefundOrder(c.Request.Context(), payments.RefundOrderInput{
		OrderID:        id,
		ActorUserID:    u.ID,
		IdempotencyKey: idem,
		AmountCents:    0,
		Reason:         note,
	})
	if err != nil {
		vm := pages.AdminOrderRefundVM{
			OrderID:   id,
			CSRFToken: middleware.GetCSRFToken(c),
			Error:     err.Error(),
		}
		render.Component(c, http.StatusBadRequest, pages.AdminOrderRefund(vm))
		return
	}

	_ = res
	c.Redirect(http.StatusSeeOther, "/admin/orders/"+id+"?refunded=1")
}

func (h *OrdersHandler) CreateShipmentLabel(c *gin.Context) {
	if h.ShippingSvc == nil {
		c.Error(apperr.Wrap(errors.New("kargo entegrasyonu devre dışı")))
		return
	}
	id := c.Param("id")

	u, ok := middleware.CurrentUser(c)
	if !ok {
		c.Error(apperr.ForbiddenErr("Giriş gerekli."))
		return
	}
	if c.PostForm("confirm") != "1" {
		render.RedirectWithFlash(c, h.Flash, "/admin/orders/"+id, view.FlashWarning, "Onay gerekli.")
		return
	}

	carrier := strings.TrimSpace(c.PostForm("carrier"))
	service := strings.TrimSpace(c.PostForm("service"))
	note := strings.TrimSpace(c.PostForm("note"))

	_, err := h.ShippingSvc.QueueShipment(c.Request.Context(), shipping.QueueShipmentInput{
		OrderID:     id,
		ActorUserID: u.ID,
		Carrier:     carrier,
		Service:     service,
		Note:        note,
	})
	if err != nil {
		msg := friendlyShipmentErr(err)
		render.RedirectWithFlash(c, h.Flash, "/admin/orders/"+id, view.FlashError, msg)
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/orders/"+id, view.FlashSuccess, "Kargo etiketi kuyruğa alındı.")
}

func (h *OrdersHandler) CreateManualShipment(c *gin.Context) {
	if h.ShippingSvc == nil {
		c.Error(apperr.Wrap(errors.New("kargo entegrasyonu devre dışı")))
		return
	}
	id := c.Param("id")

	u, ok := middleware.CurrentUser(c)
	if !ok {
		c.Error(apperr.ForbiddenErr("Giriş gerekli."))
		return
	}
	if c.PostForm("confirm") != "1" {
		render.RedirectWithFlash(c, h.Flash, "/admin/orders/"+id, view.FlashWarning, "Onay gerekli.")
		return
	}

	carrier := strings.TrimSpace(c.PostForm("carrier"))
	tracking := strings.TrimSpace(c.PostForm("tracking_no"))
	trackingURL := strings.TrimSpace(c.PostForm("tracking_url"))
	note := strings.TrimSpace(c.PostForm("note"))

	_, err := h.ShippingSvc.CreateManualShipment(c.Request.Context(), shipping.ManualShipmentInput{
		OrderID:     id,
		ActorUserID: u.ID,
		Carrier:     carrier,
		TrackingNo:  tracking,
		TrackingURL: trackingURL,
		Note:        note,
	})
	if err != nil {
		msg := friendlyShipmentErr(err)
		render.RedirectWithFlash(c, h.Flash, "/admin/orders/"+id, view.FlashError, msg)
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/orders/"+id, view.FlashSuccess, "Kargo kaydı oluşturuldu.")
}

func friendlyShipmentErr(err error) string {
	switch {
	case errors.Is(err, shipping.ErrCarrierRequired):
		return "Kargo firması zorunlu."
	case errors.Is(err, shipping.ErrTrackingRequired):
		return "Takip numarası zorunlu."
	case errors.Is(err, shipping.ErrOrderNotShippable):
		return "Sipariş kargo için uygun durumda değil."
	case errors.Is(err, shipping.ErrProviderUnavailable):
		return "Kargo sağlayıcısına ulaşılamadı."
	case errors.Is(err, shipping.ErrActorRequired):
		return "İşlem yapan kullanıcı bulunamadı."
	default:
		return "Kargo işlemi başarısız: " + err.Error()
	}
}
