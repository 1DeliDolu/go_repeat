package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/http/validation"
	cartmod "pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/internal/modules/checkout"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type CheckoutHandler struct {
	DB      *gorm.DB
	Flash   *flash.Codec
	CartCK  *cartcookie.Codec
	OrderSv *orders.Service
}

func NewCheckoutHandler(db *gorm.DB, fl *flash.Codec, ck *cartcookie.Codec, osvc *orders.Service) *CheckoutHandler {
	return &CheckoutHandler{DB: db, Flash: fl, CartCK: ck, OrderSv: osvc}
}

type checkoutInput struct {
	Email string `form:"email" binding:"omitempty,email,max=255"`

	FirstName  string `form:"first_name" binding:"required,min=2,max=100"`
	LastName   string `form:"last_name" binding:"required,min=2,max=100"`
	Address1   string `form:"address1" binding:"required,min=5,max=255"`
	Address2   string `form:"address2" binding:"omitempty,max=255"`
	City       string `form:"city" binding:"required,min=2,max=100"`
	PostalCode string `form:"postal_code" binding:"required,min=2,max=32"`
	Country    string `form:"country" binding:"required,min=2,max=2"`
	Phone      string `form:"phone" binding:"required,min=5,max=32"`

	ShippingMethod string `form:"shipping_method" binding:"required,oneof=standard express"`
	IdemKey        string `form:"idempotency_key" binding:"omitempty,max=64"`
}

type addressJSON struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2,omitempty"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	Phone      string `json:"phone"`
}

func (h *CheckoutHandler) Get(c *gin.Context) {
	u, authed := middleware.CurrentUser(c)

	cartID, ok := h.resolveCartID(c)
	if !ok {
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
		return
	}

	summary, itemsCount, currency, err := h.cartSummary(c, cartID)
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}
	if itemsCount == 0 {
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
		return
	}

	idem := randHex(16)
	form := view.CheckoutForm{
		ShippingMethod: "standard",
		IdemKey:        idem,
	}
	if authed {
		form.Email = u.Email
	}

	opts := shippingOptions(currency)
	render.Component(c, http.StatusOK, pages.Checkout(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		form,
		nil,
		"",
		opts,
		summary,
		authed,
	))
}

func (h *CheckoutHandler) Post(c *gin.Context) {
	u, authed := middleware.CurrentUser(c)

	cartID, ok := h.resolveCartID(c)
	if !ok {
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
		return
	}

	summary, itemsCount, currency, err := h.cartSummary(c, cartID)
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}
	if itemsCount == 0 {
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
		return
	}

	var in checkoutInput
	if err := c.ShouldBind(&in); err != nil {
		errs := validation.FromBindError(err, &in)
		h.renderCheckoutWithErrors(c, authed, summary, currency, errs, "", in)
		return
	}

	if !authed && strings.TrimSpace(in.Email) == "" {
		errs := validation.FieldErrors{"email": "Email zorunludur."}
		h.renderCheckoutWithErrors(c, authed, summary, currency, errs, "", in)
		return
	}

	shipCents := shippingCents(in.ShippingMethod)

	addr := addressJSON{
		FirstName:  strings.TrimSpace(in.FirstName),
		LastName:   strings.TrimSpace(in.LastName),
		Address1:   strings.TrimSpace(in.Address1),
		Address2:   strings.TrimSpace(in.Address2),
		City:       strings.TrimSpace(in.City),
		PostalCode: strings.TrimSpace(in.PostalCode),
		Country:    strings.ToUpper(strings.TrimSpace(in.Country)),
		Phone:      strings.TrimSpace(in.Phone),
	}
	addrBytes, err := json.Marshal(addr)
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	var userID *string
	var guestEmail *string
	if authed {
		userID = &u.ID
	} else {
		em := strings.ToLower(strings.TrimSpace(in.Email))
		guestEmail = &em
	}

	idem := strings.TrimSpace(in.IdemKey)
	if idem == "" {
		idem = randHex(16)
	}
	idemKey := &idem
	if !authed {
		idemKey = nil
	}

	res, err := h.OrderSv.CreateFromCart(c.Request.Context(), orders.CreateFromCartInput{
		CartID:              cartID,
		UserID:              userID,
		GuestEmail:          guestEmail,
		IdempotencyKey:      idemKey,
		TaxCents:            0,
		ShippingCents:       shipCents,
		DiscountCents:       0,
		ShippingAddressJSON: addrBytes,
		BillingAddressJSON:  nil,
	})
	if err != nil {
		var oos *checkout.OutOfStockError
		if errors.As(err, &oos) {
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Bazı ürünler stokta yok. Lütfen sepeti güncelleyin.")
			return
		}
		if errors.Is(err, orders.ErrCartEmpty) {
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
			return
		}
		h.renderCheckoutWithErrors(c, authed, summary, currency, nil, "Checkout başarısız. Lütfen tekrar deneyin.", in)
		return
	}

	if !authed {
		h.CartCK.Clear(c)
	}

	// Clear session cart cache (forces refresh on next request)
	middleware.ClearSessionCartCache(c)

	render.RedirectWithFlash(c, h.Flash, "/orders/"+res.OrderID, view.FlashSuccess, "Sipariş oluşturuldu.")
}

// --- helpers ---

func (h *CheckoutHandler) resolveCartID(c *gin.Context) (string, bool) {
	if u, ok := middleware.CurrentUser(c); ok {
		crt, err := cartmod.NewRepo(h.DB).GetOrCreateUserCart(c.Request.Context(), u.ID)
		if err == nil {
			return crt.ID, true
		}
	}
	if id, ok := h.CartCK.GetCartID(c); ok {
		return id, true
	}
	return "", false
}

func (h *CheckoutHandler) cartSummary(c *gin.Context, cartID string) (view.CheckoutSummary, int, string, error) {
	repo := cartmod.NewRepo(h.DB)
	crt, err := repo.GetCart(c.Request.Context(), cartID)
	if err != nil {
		return view.CheckoutSummary{}, 0, "EUR", err
	}

	subtotalCents := 0
	items := 0
	cur := "EUR"

	for _, it := range crt.Items {
		v := it.Variant
		cur = v.Currency
		items += it.Quantity
		subtotalCents += v.PriceCents * it.Quantity
	}

	ship := shippingCents("standard")
	total := subtotalCents + ship

	return view.CheckoutSummary{
		Currency: cur,
		Subtotal: view.MoneyFromCents(subtotalCents, cur),
		Shipping: view.MoneyFromCents(ship, cur),
		Total:    view.MoneyFromCents(total, cur),
		Items:    items,
	}, items, cur, nil
}

func (h *CheckoutHandler) renderCheckoutWithErrors(c *gin.Context, authed bool, summary view.CheckoutSummary, currency string, errs validation.FieldErrors, pageErr string, in checkoutInput) {
	form := view.CheckoutForm{
		Email:          in.Email,
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		Address1:       in.Address1,
		Address2:       in.Address2,
		City:           in.City,
		PostalCode:     in.PostalCode,
		Country:        in.Country,
		Phone:          in.Phone,
		ShippingMethod: in.ShippingMethod,
		IdemKey:        in.IdemKey,
	}
	if form.ShippingMethod == "" {
		form.ShippingMethod = "standard"
	}
	if form.IdemKey == "" {
		form.IdemKey = randHex(16)
	}

	ship := shippingCents(form.ShippingMethod)
	summary.Shipping = view.MoneyFromCents(ship, currency)

	opts := shippingOptions(currency)

	render.Component(c, http.StatusBadRequest, pages.Checkout(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		form,
		errs,
		pageErr,
		opts,
		summary,
		authed,
	))
}

func shippingOptions(currency string) []view.ShippingOption {
	return []view.ShippingOption{
		{Code: "standard", Label: "Standard (2-4 gün)", Price: view.MoneyFromCents(shippingCents("standard"), currency)},
		{Code: "express", Label: "Express (1-2 gün)", Price: view.MoneyFromCents(shippingCents("express"), currency)},
	}
}

func shippingCents(method string) int {
	switch method {
	case "express":
		return 1500
	default:
		return 500
	}
}
