package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/http/validation"
	cartmod "pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/internal/modules/checkout"
	emailmod "pehlione.com/app/internal/modules/email"
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
	EmailSv *emailmod.OutboxService
	BaseURL string
}

func NewCheckoutHandler(db *gorm.DB, fl *flash.Codec, ck *cartcookie.Codec, osvc *orders.Service, emailSvc *emailmod.OutboxService, baseURL string) *CheckoutHandler {
	return &CheckoutHandler{DB: db, Flash: fl, CartCK: ck, OrderSv: osvc, EmailSv: emailSvc, BaseURL: baseURL}
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
	}

	opts := shippingOptions(currency)
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
	svc := cartmod.NewService(h.DB)

	var cartPage view.CartPage
	var err error

	if u, ok := middleware.CurrentUser(c); ok {
		// Logged-in user
		cartPage, err = svc.BuildCartPageForUser(c.Request.Context(), u.ID)
	} else {
		// Guest user
		cc, _ := h.CartCK.Get(c)
		cartPage, err = svc.BuildCartPageFromCookie(c.Request.Context(), cc)
	}

	if err != nil {
		return view.CheckoutSummary{}, 0, "EUR", err
	}

	ship := shippingCents("standard")
	total := cartPage.SubtotalCents + ship

	return view.CheckoutSummary{
		Currency:      cartPage.Currency,
		Subtotal:      cartPage.Subtotal,
		Shipping:      view.MoneyFromCents(ship, cartPage.Currency),
		Total:         view.MoneyFromCents(total, cartPage.Currency),
		Items:         cartPage.Count,
		Lines:         cartPage.Items,
		SubtotalCents: cartPage.SubtotalCents,
		ShippingCents: ship,
		TotalCents:    total,
	}, cartPage.Count, cartPage.Currency, nil
}

func (h *CheckoutHandler) createTempCartFromCookie(c *gin.Context, cc *cartcookie.Cart) (*cartmod.Cart, error) {
	repo := cartmod.NewRepo(h.DB)

	// Create empty cart with UUID
	tempCart := cartmod.Cart{
		ID:     uuid.NewString(),
		UserID: nil,
	}
	if err := h.DB.Create(&tempCart).Error; err != nil {
		log.Printf("createTempCartFromCookie: failed to create cart: %v", err)
		return nil, err
	}

	// Add items from cookie
	for _, it := range cc.Items {
		if it.VariantID == "" || it.Qty <= 0 {
			continue
		}
		if err := repo.AddItem(c.Request.Context(), tempCart.ID, it.VariantID, it.Qty); err != nil {
			log.Printf("createTempCartFromCookie: failed to add item %s: %v", it.VariantID, err)
			return nil, err
		}
	}

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
	summary.Shipping = view.MoneyFromCents(ship, currency)
	summary.ShippingCents = ship
	summary.TotalCents = summary.SubtotalCents + ship
	summary.Total = view.MoneyFromCents(summary.TotalCents, currency)

	opts := shippingOptions(currency)
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

	addressLines := formatAddressLines(addr)
	pdfURL := strings.TrimRight(h.BaseURL, "/") + "/orders/" + orderID + "/invoice.pdf"

	subject := fmt.Sprintf("Siparişiniz alındı (#%s)", orderID)
	htmlBody := buildConfirmationHTML(order, items, addr, addressLines, pdfURL)
	textBody := buildConfirmationText(order, items, addr, addressLines, pdfURL)

	if err := h.EmailSv.Enqueue(ctx, to, subject, textBody, htmlBody); err != nil {
		log.Printf("checkout: failed to enqueue confirmation email: %v", err)
	}
}

func formatAddressLines(addr addressJSON) []string {
	var lines []string
	fullName := strings.TrimSpace(strings.TrimSpace(addr.FirstName) + " " + strings.TrimSpace(addr.LastName))
	if fullName != "" {
		lines = append(lines, fullName)
	}
	if addr.Address1 != "" {
		lines = append(lines, addr.Address1)
	}
	if addr.Address2 != "" {
		lines = append(lines, addr.Address2)
	}
	cityLine := strings.TrimSpace(strings.TrimSpace(addr.PostalCode) + " " + addr.City)
	if cityLine != "" {
		lines = append(lines, cityLine)
	}
	if addr.Country != "" {
		lines = append(lines, strings.ToUpper(addr.Country))
	}
	if addr.Phone != "" {
		lines = append(lines, "Tel: "+addr.Phone)
	}
	return lines
}

func buildConfirmationHTML(order orders.Order, items []orders.OrderItem, addr addressJSON, addressLines []string, pdfURL string) string {
	brandYellow := "#FACC15"
	brandOrange := "#F97316"
	var sb strings.Builder
	sb.WriteString(`<div style="background-color:#f8fafc;padding:24px;font-family:'Inter','Segoe UI',Arial,sans-serif;">`)
	sb.WriteString(`<div style="max-width:640px;margin:0 auto;background:#ffffff;border-radius:16px;padding:32px;box-shadow:0 12px 24px rgba(15,23,42,0.08);">`)
	sb.WriteString(`<div style="text-align:center;margin-bottom:24px;">`)
	sb.WriteString(`<span style="font-size:28px;font-weight:700;letter-spacing:1px;color:` + brandYellow + `">pehli</span>`)
	sb.WriteString(`<span style="font-size:28px;font-weight:700;letter-spacing:1px;color:` + brandOrange + `">ONE</span>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`<p style="font-size:16px;color:#0f172a;margin-bottom:8px;">Merhaba,</p>`)
	sb.WriteString(`<p style="font-size:15px;color:#475569;margin-bottom:24px;">Siparişiniz başarıyla alındı. Detaylar aşağıdadır.</p>`)
	sb.WriteString(`<div style="border:1px solid #e2e8f0;border-radius:12px;padding:16px;margin-bottom:24px;">`)
	sb.WriteString(`<p style="margin:0;font-size:14px;color:#475569;"><strong>Sipariş No:</strong> ` + html.EscapeString(order.ID) + `</p>`)
	sb.WriteString(`<p style="margin:4px 0;font-size:14px;color:#475569;"><strong>Ödeme:</strong> ` + html.EscapeString(view.PaymentMethodLabel(addr.PaymentMethod)) + `</p>`)
	sb.WriteString(`<p style="margin:4px 0;font-size:14px;color:#475569;"><strong>Teslimat:</strong> ` + html.EscapeString(view.ShippingLabel(addr.ShippingMethod)) + `</p>`)
	if len(addressLines) > 0 {
		sb.WriteString(`<p style="margin:8px 0 0;font-size:14px;color:#475569;"><strong>Adres:</strong><br/>` + html.EscapeString(strings.Join(addressLines, ", ")) + `</p>`)
	}
	sb.WriteString(`</div>`)

	sb.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:14px;color:#0f172a;">`)
	sb.WriteString(`<thead><tr style="background:#f8fafc;">`)
	sb.WriteString(`<th style="padding:8px;border:1px solid #e2e8f0;text-align:left;">Ürün</th>`)
	sb.WriteString(`<th style="padding:8px;border:1px solid #e2e8f0;text-align:center;">Adet</th>`)
	sb.WriteString(`<th style="padding:8px;border:1px solid #e2e8f0;text-align:right;">Tutar</th>`)
	sb.WriteString(`</tr></thead><tbody>`)
	for _, it := range items {
		sb.WriteString(`<tr>`)
		sb.WriteString(`<td style="padding:8px;border:1px solid #e2e8f0;">` + html.EscapeString(it.ProductName) + `</td>`)
		sb.WriteString(fmt.Sprintf(`<td style="padding:8px;border:1px solid #e2e8f0;text-align:center;">%d</td>`, it.Quantity))
		sb.WriteString(`<td style="padding:8px;border:1px solid #e2e8f0;text-align:right;">` + view.MoneyFromCents(it.LineTotalCents, it.Currency) + `</td>`)
		sb.WriteString(`</tr>`)
	}
	sb.WriteString(`</tbody></table>`)

	sb.WriteString(`<div style="margin-top:24px;border-top:1px solid #e2e8f0;padding-top:16px;">`)
	sb.WriteString(`<p style="display:flex;justify-content:space-between;font-size:14px;margin:4px 0;color:#475569;"><span>Ara Toplam</span><strong>` + view.MoneyFromCents(order.SubtotalCents, order.Currency) + `</strong></p>`)
	sb.WriteString(`<p style="display:flex;justify-content:space-between;font-size:14px;margin:4px 0;color:#475569;"><span>Kargo</span><strong>` + view.MoneyFromCents(order.ShippingCents, order.Currency) + `</strong></p>`)
	sb.WriteString(`<p style="display:flex;justify-content:space-between;font-size:16px;margin:8px 0;color:#0f172a;"><span>Toplam</span><strong>` + view.MoneyFromCents(order.TotalCents, order.Currency) + `</strong></p>`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<div style="margin-top:24px;text-align:center;">`)
	sb.WriteString(`<a href="` + html.EscapeString(pdfURL) + `" style="display:inline-flex;align-items:center;gap:8px;background:` + brandOrange + `;color:#fff;padding:12px 24px;border-radius:999px;text-decoration:none;font-weight:600;">PDF Fatura İndir</a>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`<p style="font-size:13px;color:#94a3b8;margin-top:24px;text-align:center;">pehliONE · Bu e-posta bilgilendirme amaçlıdır.</p>`)
	sb.WriteString(`</div></div>`)
	return sb.String()
}

func buildConfirmationText(order orders.Order, items []orders.OrderItem, addr addressJSON, addressLines []string, pdfURL string) string {
	var sb strings.Builder
	sb.WriteString("pehliONE Sipariş Onayı\n")
	sb.WriteString("--------------------------------\n")
	sb.WriteString(fmt.Sprintf("Sipariş No: %s\n", order.ID))
	sb.WriteString(fmt.Sprintf("Ödeme: %s\n", view.PaymentMethodLabel(addr.PaymentMethod)))
	sb.WriteString(fmt.Sprintf("Teslimat: %s\n", view.ShippingLabel(addr.ShippingMethod)))
	if len(addressLines) > 0 {
		sb.WriteString("Adres:\n")
		for _, line := range addressLines {
			sb.WriteString("  " + line + "\n")
		}
	}
	sb.WriteString("\nÜrünler:\n")
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("- %s x%d: %s\n", it.ProductName, it.Quantity, view.MoneyFromCents(it.LineTotalCents, it.Currency)))
	}
	sb.WriteString("\nAra Toplam: " + view.MoneyFromCents(order.SubtotalCents, order.Currency) + "\n")
	sb.WriteString("Kargo: " + view.MoneyFromCents(order.ShippingCents, order.Currency) + "\n")
	sb.WriteString("Toplam: " + view.MoneyFromCents(order.TotalCents, order.Currency) + "\n")
	sb.WriteString("\nPDF fatura: " + pdfURL + "\n")
	sb.WriteString("\nTeşekkürler,\npehliONE\n")
	return sb.String()
}
