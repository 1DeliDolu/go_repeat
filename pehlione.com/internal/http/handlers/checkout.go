package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"pehlione.com/app/internal/emails"
	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/http/validation"
	cartmod "pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/internal/modules/checkout"
	"pehlione.com/app/internal/modules/currency"
	emailmod "pehlione.com/app/internal/modules/email"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type CheckoutHandler struct {
	DB          *gorm.DB
	Flash       *flash.Codec
	CartCK      *cartcookie.Codec
	CartSvc     *cartmod.Service
	OrderSv     *orders.Service
	EmailSv     *emailmod.OutboxService
	CurrencySvc *currency.Service
	BaseURL     string
}

func NewCheckoutHandler(db *gorm.DB, fl *flash.Codec, ck *cartcookie.Codec, cartSvc *cartmod.Service, osvc *orders.Service, emailSvc *emailmod.OutboxService, currSvc *currency.Service, baseURL string) *CheckoutHandler {
	return &CheckoutHandler{
		DB:          db,
		Flash:       fl,
		CartCK:      ck,
		CartSvc:     cartSvc,
		OrderSv:     osvc,
		EmailSv:     emailSvc,
		CurrencySvc: currSvc,
		BaseURL:     baseURL,
	}
}

type checkoutInput struct {
	Email string `form:"email" binding:"omitempty,email,max=255"`

	FirstName  string `form:"first_name" binding:"required,min=2,max=100"`
	LastName   string `form:"last_name" binding:"required,min=2,max=100"`
	Address1   string `form:"address1" binding:"required,min=5,max=255"`
	Address2   string `form:"address2" binding:"omitempty,max=255"`
	City       string `form:"city" binding:"required,min=2,max=100"`
	PostalCode string `form:"postal_code" binding:"required,min=2,max=32"`
	Country    string `form:"country" binding:"required,len=2"`
	Phone      string `form:"phone" binding:"required,min=5,max=32"`

	ShippingMethod string `form:"shipping_method" binding:"required,oneof=standard express"`
	PaymentMethod  string `form:"payment_method" binding:"required,oneof=card paypal klarna"`
	IdemKey        string `form:"idempotency_key" binding:"omitempty,max=64"`
}

type addressJSON struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Address1       string `json:"address1"`
	Address2       string `json:"address2,omitempty"`
	City           string `json:"city"`
	PostalCode     string `json:"postal_code"`
	Country        string `json:"country"`
	Phone          string `json:"phone"`
	ShippingMethod string `json:"shipping_method,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`
}

func (h *CheckoutHandler) Get(c *gin.Context) {
	u, authed := middleware.CurrentUser(c)

	summary, itemsCount, currency, err := h.buildCartSummary(c)
	if err != nil {
		log.Printf("Checkout GET: buildCartSummary failed for user %v: %v", u.ID, err)
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
		// Pre-fill address information for authenticated users
		if u.FirstName != nil {
			form.FirstName = *u.FirstName
		}
		if u.LastName != nil {
			form.LastName = *u.LastName
		}
		if u.Address != nil {
			form.Address1 = *u.Address
		}
		if u.PhoneE164 != nil {
			form.Phone = *u.PhoneE164
		}
		log.Printf("Checkout GET: pre-filled form for user %s (email=%s)", u.ID, u.Email)
	}

	opts := shippingOptions(c.Request.Context(), h.CurrencySvc, currency)
	payments := paymentOptions()
	render.Component(c, http.StatusOK, pages.Checkout(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		form,
		nil,
		"",
		opts,
		payments,
		summary,
		authed,
	))
}

func (h *CheckoutHandler) Post(c *gin.Context) {
	u, authed := middleware.CurrentUser(c)

	summary, itemsCount, currency, err := h.buildCartSummary(c)
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
	summary.ShippingCents = shipCents
	summary.Shipping = view.MoneyFromCents(shipCents, currency)
	summary.TotalCents = summary.SubtotalCents + shipCents
	summary.Total = view.MoneyFromCents(summary.TotalCents, currency)

	addr := addressJSON{
		FirstName:      strings.TrimSpace(in.FirstName),
		LastName:       strings.TrimSpace(in.LastName),
		Address1:       strings.TrimSpace(in.Address1),
		Address2:       strings.TrimSpace(in.Address2),
		City:           strings.TrimSpace(in.City),
		PostalCode:     strings.TrimSpace(in.PostalCode),
		Country:        strings.ToUpper(strings.TrimSpace(in.Country)),
		Phone:          strings.TrimSpace(in.Phone),
		ShippingMethod: in.ShippingMethod,
		PaymentMethod:  in.PaymentMethod,
	}
	addrBytes, err := json.Marshal(addr)
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	var userID *string
	var guestEmail *string
	var cartID string

	if authed {
		userID = &u.ID
		log.Printf("Checkout: authenticated user %s (email=%s) proceeding with checkout", u.ID, u.Email)
		// Get user cart ID
		crt, err := cartmod.NewRepo(h.DB).GetOrCreateUserCart(c.Request.Context(), u.ID)
		if err != nil {
			middleware.Fail(c, apperr.Wrap(err))
			return
		}
		cartID = crt.ID
	} else {
		em := strings.ToLower(strings.TrimSpace(in.Email))
		guestEmail = &em
		log.Printf("Checkout: guest checkout with email=%s", em)

		// Guest: create temporary cart from cookie
		cc, _ := h.CartCK.Get(c)
		if cc == nil || len(cc.Items) == 0 {
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
			return
		}

		// Create temp cart in DB for order creation
		tempCart, err := h.createTempCartFromCookie(c, cc)
		if err != nil {
			middleware.Fail(c, apperr.Wrap(err))
			return
		}
		cartID = tempCart.ID
	}

	idem := strings.TrimSpace(in.IdemKey)
	if idem == "" {
		idem = randHex(16)
	}
	idemKey := &idem
	if !authed {
		idemKey = nil
	}

	log.Printf("Creating order: cartID=%s, userID=%v, guestEmail=%v, shipCents=%d", cartID, userID, guestEmail, shipCents)

	chargeCurrency := currency
	if h.CurrencySvc != nil {
		chargeCurrency = h.CurrencySvc.ChooseChargeCurrency(currency)
	}

	log.Printf("Starting CreateFromCart: cartID=%s", cartID)
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
		DisplayCurrency:     currency,
		ChargeCurrency:      chargeCurrency,
	})
	if err != nil {
		log.Printf("CreateFromCart failed with error type %T: %v", err, err)
		var oos *checkout.OutOfStockError
		if errors.As(err, &oos) {
			log.Printf("Checkout failed: out of stock - %v", err)
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Bazı ürünler stokta yok. Lütfen sepeti güncelleyin.")
			return
		}
		if errors.Is(err, orders.ErrCartEmpty) {
			log.Printf("Checkout failed: cart empty")
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet boş.")
			return
		}
		if errors.Is(err, orders.ErrProductUnavailable) {
			log.Printf("Checkout failed: product unavailable")
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Bazı ürünler mevcut değil.")
			return
		}
		if errors.Is(err, orders.ErrCurrencyMismatch) {
			log.Printf("Checkout failed: currency mismatch")
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Para birimi uyuşmazlığı.")
			return
		}
		log.Printf("Checkout error (unhandled): %T - %v", err, err)
		h.renderCheckoutWithErrors(c, authed, summary, currency, nil, "Checkout başarısız. Lütfen tekrar deneyin.", in)
		return
	}

	if !authed {
		h.CartCK.Clear(c)
	}

	// Clear session cart cache (forces refresh on next request)
	middleware.ClearSessionCartCache(c)

	if h.EmailSv != nil {
		emailAddr := ""
		if authed {
			emailAddr = strings.TrimSpace(u.Email)
		} else {
			emailAddr = strings.TrimSpace(in.Email)
		}
		h.sendOrderConfirmation(c.Request.Context(), emailAddr, res.OrderID, in.PaymentMethod, in.ShippingMethod)
	}

	render.RedirectWithFlash(c, h.Flash, "/orders/"+res.OrderID, view.FlashSuccess, "Sipariş oluşturuldu.")
}

// --- helpers ---

func (h *CheckoutHandler) buildCartSummary(c *gin.Context) (view.CheckoutSummary, int, string, error) {
	svc := h.CartSvc
	if svc == nil {
		svc = cartmod.NewService(h.DB, h.CurrencySvc)
	}
	displayCurrency := middleware.GetDisplayCurrency(c)
	var cartPage view.CartPage
	var err error

	if u, ok := middleware.CurrentUser(c); ok {
		// Logged-in user
		cartPage, err = svc.BuildCartPageForUser(c.Request.Context(), u.ID, displayCurrency)
	} else {
		// Guest user
		cc, _ := h.CartCK.Get(c)
		cartPage, err = svc.BuildCartPageFromCookie(c.Request.Context(), cc, displayCurrency)
	}

	if err != nil {
		return view.CheckoutSummary{}, 0, displayCurrency, err
	}

	ship := shippingCents("standard")
	shipDisplay := ship
	if h.CurrencySvc != nil {
		if converted, _, convErr := h.CurrencySvc.ConvertDisplay(c.Request.Context(), ship, cartPage.Currency); convErr == nil {
			shipDisplay = converted
		}
	}
	displayTotal := cartPage.DisplayTotalCents + shipDisplay
	totalBase := cartPage.BaseSubtotalCents + ship

	return view.CheckoutSummary{
		Currency:             cartPage.Currency,
		BaseCurrency:         cartPage.BaseCurrency,
		Subtotal:             cartPage.Subtotal,
		Shipping:             view.MoneyFromCents(shipDisplay, cartPage.Currency),
		Total:                view.MoneyFromCents(displayTotal, cartPage.Currency),
		Items:                cartPage.Count,
		Lines:                cartPage.Items,
		SubtotalCents:        cartPage.BaseSubtotalCents,
		ShippingCents:        ship,
		TotalCents:           totalBase,
		BaseSubtotalCents:    cartPage.BaseSubtotalCents,
		BaseShippingCents:    ship,
		BaseTotalCents:       totalBase,
		DisplaySubtotalCents: cartPage.DisplaySubtotalCents,
		DisplayShippingCents: shipDisplay,
		DisplayTotalCents:    displayTotal,
	}, cartPage.Count, cartPage.Currency, nil
}

func (h *CheckoutHandler) createTempCartFromCookie(c *gin.Context, cc *cartcookie.Cart) (*cartmod.Cart, error) {
	log.Printf("createTempCartFromCookie: starting with %d items from cookie", len(cc.Items))
	repo := cartmod.NewRepo(h.DB)

	// Create empty cart with UUID
	tempCart := cartmod.Cart{
		ID:     uuid.NewString(),
		UserID: nil,
	}
	log.Printf("createTempCartFromCookie: creating temp cart with ID %s", tempCart.ID)
	if err := h.DB.Create(&tempCart).Error; err != nil {
		log.Printf("createTempCartFromCookie: failed to create cart: %v", err)
		return nil, err
	}

	// Add items from cookie
	addedCount := 0
	for _, it := range cc.Items {
		if it.VariantID == "" || it.Qty <= 0 {
			log.Printf("createTempCartFromCookie: skipping invalid item variant_id=%s qty=%d", it.VariantID, it.Qty)
			continue
		}
		log.Printf("createTempCartFromCookie: adding item to cart variant_id=%s qty=%d", it.VariantID, it.Qty)
		if err := repo.AddItem(c.Request.Context(), tempCart.ID, it.VariantID, it.Qty); err != nil {
			log.Printf("createTempCartFromCookie: failed to add item %s: %v", it.VariantID, err)
			return nil, err
		}
		addedCount++
	}

	log.Printf("createTempCartFromCookie: successfully created temp cart %s with %d items", tempCart.ID, addedCount)
	return &tempCart, nil
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
		PaymentMethod:  in.PaymentMethod,
		IdemKey:        in.IdemKey,
	}
	if form.ShippingMethod == "" {
		form.ShippingMethod = "standard"
	}
	if form.PaymentMethod == "" {
		form.PaymentMethod = "card"
	}
	if form.IdemKey == "" {
		form.IdemKey = randHex(16)
	}

	ship := shippingCents(form.ShippingMethod)
	shipDisplay := ship
	if h.CurrencySvc != nil {
		if converted, _, err := h.CurrencySvc.ConvertDisplay(c.Request.Context(), ship, currency); err == nil {
			shipDisplay = converted
		}
	}
	summary.Shipping = view.MoneyFromCents(shipDisplay, currency)
	summary.ShippingCents = ship
	summary.BaseShippingCents = ship
	summary.DisplayShippingCents = shipDisplay
	summary.TotalCents = summary.SubtotalCents + ship
	displayTotal := summary.DisplaySubtotalCents + shipDisplay
	summary.DisplayTotalCents = displayTotal
	summary.Total = view.MoneyFromCents(displayTotal, currency)

	opts := shippingOptions(c.Request.Context(), h.CurrencySvc, currency)
	payments := paymentOptions()

	render.Component(c, http.StatusBadRequest, pages.Checkout(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		form,
		errs,
		pageErr,
		opts,
		payments,
		summary,
		authed,
	))
}

func shippingOptions(ctx context.Context, currSvc *currency.Service, currency string) []view.ShippingOption {
	return []view.ShippingOption{
		{Code: "standard", Label: "Standard (2-4 gün)", Price: formatShippingPrice(ctx, currSvc, shippingCents("standard"), currency)},
		{Code: "express", Label: "Express (1-2 gün)", Price: formatShippingPrice(ctx, currSvc, shippingCents("express"), currency)},
	}
}

func formatShippingPrice(ctx context.Context, currSvc *currency.Service, base int, currency string) string {
	amount := base
	if currSvc != nil {
		if converted, _, err := currSvc.ConvertDisplay(ctx, base, currency); err == nil {
			amount = converted
		}
	}
	return view.MoneyFromCents(amount, currency)
}

func shippingCents(method string) int {
	switch method {
	case "express":
		return 1500
	default:
		return 500
	}
}

func paymentOptions() []view.PaymentOption {
	return []view.PaymentOption{
		{Code: "card", Label: "Kart (Visa / Mastercard)", Description: "3D Secure destekli kredi veya banka kartınız ile güvenli ödeme."},
		{Code: "paypal", Label: "PayPal", Description: "PayPal hesabınızla hızlı ödeme yapın."},
		{Code: "klarna", Label: "Klarna \"Pay Later\"", Description: "Şimdi al, 30 gün sonra öde seçeneği."},
	}
}

func (h *CheckoutHandler) sendOrderConfirmation(ctx context.Context, emailAddr string, orderID string, paymentMethod, shippingMethod string) {
	if h.EmailSv == nil {
		return
	}
	to := strings.TrimSpace(emailAddr)
	if to == "" {
		return
	}

	orderRepo := orders.NewRepo(h.DB)
	order, items, err := orderRepo.GetWithItems(ctx, orderID)
	if err != nil {
		log.Printf("checkout: confirmation skipped (order fetch failed): %v", err)
		return
	}

	var addr addressJSON
	if len(order.ShippingAddressJSON) > 0 {
		_ = json.Unmarshal(order.ShippingAddressJSON, &addr)
	}
	if addr.ShippingMethod == "" {
		addr.ShippingMethod = shippingMethod
	}
	if addr.PaymentMethod == "" {
		addr.PaymentMethod = paymentMethod
	}

	payload := emails.BuildOrderPayload(h.BaseURL, order, items, "Paid", "")
	payload["PreviewText"] = "Payment received - here are your order details."

	if err := h.EmailSv.Enqueue(ctx, emailmod.Job{
		To:       to,
		Template: emailmod.TemplateOrderPaid,
		Payload:  payload,
	}); err != nil {
		log.Printf("checkout: failed to enqueue confirmation email: %v", err)
	}
}
