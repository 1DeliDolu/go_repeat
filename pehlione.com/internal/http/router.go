package http

import (
	"context"
	"crypto/rand"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/config"
	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/handlers"
	adminHandlers "pehlione.com/app/internal/http/handlers/admin"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/mailer"
	"pehlione.com/app/internal/modules/auth"
	"pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/internal/modules/currency"
	"pehlione.com/app/internal/modules/email"
	"pehlione.com/app/internal/modules/fx"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/payments"
	"pehlione.com/app/internal/modules/products"
	"pehlione.com/app/internal/modules/shipping"
	"pehlione.com/app/internal/modules/users"
	"pehlione.com/app/internal/modules/wishlist"
	"pehlione.com/app/internal/sms"
	"pehlione.com/app/internal/storage"
	"pehlione.com/app/pkg/view"
)

func NewRouter(logger *slog.Logger, db *gorm.DB, cfg config.AppConfig) *gin.Engine {
	// --- Secrets / Codecs ---
	secret := mustSecret()
	appBaseURL := cfg.AppBaseURL
	brandName := cfg.Email.FromName
	if strings.TrimSpace(brandName) == "" {
		brandName = "pehliONE"
	}
	supportEmail := cfg.Email.SMTP.From
	if strings.TrimSpace(supportEmail) == "" {
		supportEmail = "support@pehlione.com"
	}

	flashCookieName := envOr("FLASH_COOKIE_NAME", "pehlione_flash")
	flashSecure := envBool("FLASH_COOKIE_SECURE", false)
	flashCodec := flash.NewCodec(secret, flashCookieName, flashSecure)

	// CSRF (double-submit cookie)
	csrfCfg := middleware.CSRFCfg{
		CookieName: envOr("CSRF_COOKIE_NAME", "pehlione_csrf"),
		Secure:     envBool("CSRF_COOKIE_SECURE", false),
	}

	// Session
	ttlHours := envInt("SESSION_TTL_HOURS", 168)
	sessCfg := middleware.SessionCfg{
		DB:         db,
		CookieName: envOr("SESSION_COOKIE_NAME", "pehlione_session"),
		Secure:     envBool("SESSION_COOKIE_SECURE", false),
		TTL:        time.Duration(ttlHours) * time.Hour,
	}

	fxRepo := fx.NewRepo(db)
	fxSvc := fx.NewService(fxRepo, cfg.Currency.BaseCurrency)
	currencySvc := currency.NewService(fxSvc, currency.Config{
		BaseCurrency:      cfg.Currency.BaseCurrency,
		DefaultDisplay:    cfg.Currency.DefaultDisplay,
		DisplayCurrencies: cfg.Currency.DisplayCurrencies,
		ChargeCurrencies:  cfg.Currency.ChargeCurrencies,
	})

	// Cart cookie
	cartCKName := envOr("CART_COOKIE_NAME", "pehlione_cart")
	cartCKSecure := envBool("CART_COOKIE_SECURE", false)
	cartCK := cartcookie.New(secret, cartCKName, cartCKSecure)
	cartSvc := cart.NewService(db, currencySvc)

	// --- Router + Middleware order ---
	r := gin.New()
	if proxies := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); proxies != "" {
		list := strings.Split(proxies, ",")
		if err := r.SetTrustedProxies(list); err != nil {
			logger.Warn("failed to set trusted proxies", slog.String("error", err.Error()))
		}
	} else {
		r.SetTrustedProxies(nil)
	}

	// Core observability first
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	// Request-scoped UI context
	r.Use(middleware.FlashMiddleware(flashCodec))
	r.Use(middleware.CSRF(csrfCfg))
	r.Use(middleware.SessionMiddleware(sessCfg))
	r.Use(middleware.CurrencyPreference(middleware.CurrencyPrefCfg{
		Service:    currencySvc,
		CookieName: cfg.Currency.CookieName,
		Secure:     cfg.Currency.CookieSecure,
	}))

	var emailSvc *email.OutboxService

	// Cart badge: DB-backed (logged-in) + cookie fallback (guest)
	cookieQtyFunc := func(c *gin.Context) (int, error) {
		// cartCK codec'i yalnızca cart ID tutuyor, items bilgisi yok.
		// Cookie tabanlı qty için şimdi fallback olarak 0 dönüyoruz;
		// ilerde cartcookie'de item data tutmaya karar verirseniz burayı güncelleyin.
		return 0, nil
	}
	r.Use(middleware.CartBadge(middleware.CartBadgeCfg{
		DB:         db,
		CookieQty:  cookieQtyFunc,
		CartSvc:    cartSvc,
		CartCookie: cartCK,
	}))

	// Error pipeline + panic safety
	r.Use(middleware.ErrorHandler(logger))
	r.Use(middleware.Recovery(logger))

	// Static
	r.Static("/static", "./static")
	// Serve uploads when using local storage driver
	r.Static("/uploads", "./storage/uploads")

	// --- Routes ---
	r.GET("/", handlers.Home)
	r.GET("/healthz", handlers.Healthz)

	// Products (public product listing)
	productsRepo := products.NewGormRepo(db)
	productsSvc := products.NewService(productsRepo)
	wishlistSvc := wishlist.NewService(db)
	productsH := handlers.NewProductsHandler(productsSvc, currencySvc)
	r.GET("/products", productsH.List)
	r.GET("/products/:slug", productsH.Show)

	// Cart codec
	cartCodec := cartcookie.New(secret, "pehlione_cart", false) // dev: secure=false

	// Cart (public shopping cart page)
	cartH := handlers.NewCartHandler(db, flashCodec, cartCodec, cartSvc)
	r.GET("/cart", cartH.Get)
	r.POST("/cart/items", cartH.Add) // SSR: add to cart + redirect (form submission dari product pages)
	r.POST("/cart/items/update", cartH.Update)
	r.POST("/cart/items/remove", cartH.Remove)

	// Company (public company info page)
	companyH := handlers.NewCompanyHandler()
	r.GET("/company", companyH.Get)

	// Demo: flash set + redirect (POST => CSRF token gerekir)
	r.POST("/demo/flash", func(c *gin.Context) {
		render.RedirectWithFlash(c, flashCodec, "/", view.FlashSuccess, "İşlem başarılı (flash).")
	})

	currencyPrefH := handlers.NewCurrencyPreferenceHandler(currencySvc, cfg.Currency.CookieName, cfg.Currency.CookieSecure)
	r.POST("/settings/currency", currencyPrefH.Post)

	// Auth (DB-backed): signup/login/logout
	authH := handlers.NewAuthHandlers(db, flashCodec, sessCfg, cartCK)

	r.GET("/signup", authH.SignupGet)
	r.POST("/signup", authH.SignupPost)
	r.GET("/verify", authH.VerifyGet)

	verifyEmailH := handlers.NewVerifyEmailHandler(db)
	r.GET("/verify-email", verifyEmailH.Get)
	r.GET("/login", authH.LoginGet)
	r.POST("/login", authH.LoginPost)
	r.POST("/logout", authH.LogoutPost)

	// Admin (protected)
	admin := r.Group("/admin")
	admin.Use(middleware.RequireAdmin(flashCodec))
	admin.GET("", handlers.AdminDashboard)

	stRes, err := storage.FromEnv(context.Background())
	if err != nil {
		panic(err)
	}
	ph := adminHandlers.NewProductsHandler(db, flashCodec, stRes.Storage)

	admin.GET("/products", ph.List)
	admin.GET("/products/new", ph.New)
	admin.POST("/products", ph.Create)
	admin.GET("/products/:id/edit", ph.Edit)
	admin.POST("/products/:id", ph.Update)
	admin.POST("/products/:id/delete", ph.Delete)

	admin.POST("/products/:id/variants", ph.AddVariant)

	// Payment provider (used by both checkout and admin)
	var provider payments.Provider
	switch cfg.Payment.Provider {
	case "", "mock":
		provider = payments.NewMockProvider(cfg.Payment.MockWebhookSecret, cfg.Payment.MockWebhookTolerance)
	default:
		log.Fatalf("unsupported payment provider: %s", cfg.Payment.Provider)
	}

	// Webhook service + handler
	webhookSvc := payments.NewWebhookService(db)
	webhookH := handlers.NewWebhookHandler(logger, provider, webhookSvc)

	admin.POST("/products/:id/variants/:vid/delete", ph.DeleteVariant)
	admin.POST("/products/:id/variants/:vid", ph.UpdateVariant)
	admin.POST("/products/:id/variants/:vid/sku", ph.UpdateVariantSKU)

	admin.POST("/products/:id/images", ph.AddImage)
	admin.POST("/products/:id/images/:iid/delete", ph.DeleteImage)
	admin.POST("/products/:id/images/upload", ph.UploadImage)

	// Protected routes (require authentication)
	authOnly := r.Group("")
	authOnly.Use(middleware.RequireAuth(flashCodec))

	authRepo := auth.NewRepo(db)
	ordersRepo := orders.NewRepo(db)
	accountH := handlers.NewAccountHandler(authRepo, flashCodec, wishlistSvc, productsRepo, ordersRepo)
	authOnly.GET("/account", accountH.Get)
	authOnly.POST("/account/password", accountH.ChangePassword)

	// Account Edit routes
	accountEditH := handlers.NewAccountEditHandler(authRepo, flashCodec)
	authOnly.GET("/account/edit", accountEditH.Get)
	authOnly.POST("/account/edit", accountEditH.Post)

	// Account routes
	accountOrdersH := handlers.NewAccountOrdersHandler(ordersRepo, authRepo, flashCodec)
	account := r.Group("/account")
	account.Use(middleware.RequireAuth(flashCodec))
	account.GET("/orders", accountOrdersH.List)

	smsRepo := sms.NewOutboxRepository(db)
	smsH := handlers.NewSmsHandler(db, smsRepo, flashCodec, logger)

	// Set up SMS verification service if SMS provider is available
	if cfg.SMS.Enabled && cfg.SMS.Provider != "" {
		var smsProvider sms.SMSProvider
		switch cfg.SMS.Provider {
		case "mock":
			smsProvider = sms.NewMockProvider(logger)
		default:
			log.Fatalf("router: unsupported sms provider %s", cfg.SMS.Provider)
		}
		smsSvc := sms.NewVerificationService(db, smsProvider)
		smsH.SetVerificationService(smsSvc)
	}

	account.POST("/sms", smsH.PostAccountSMS)
	account.POST("/sms/verify", smsH.PostAccountSMSVerify)
	account.POST("/sms/send-code", smsH.PostSendCode)

	wishlistH := handlers.NewWishlistHandler(wishlistSvc, productsRepo, currencySvc)
	authOnly.GET("/wishlist", wishlistH.List)
	authOnly.POST("/wishlist/items", wishlistH.Add)
	authOnly.POST("/wishlist/items/remove", wishlistH.Remove)

	// --- Email worker initialization ---
	var verifyService *users.VerifyService
	if cfg.Email.Enabled {
		emailSvc = email.NewService(db)

		smtpMailer := mailer.NewSMTPMailer(cfg.Email.SMTP)
		sender := email.NewMailerAdapter(smtpMailer, cfg.Email.SMTP.From, brandName)
		log.Printf("Email worker: initializing with SMTP host=%s port=%s from=%s", cfg.Email.SMTP.Host, cfg.Email.SMTP.Port, cfg.Email.SMTP.From)

		renderer := email.NewRenderer(appBaseURL, brandName, supportEmail)
		worker := email.NewWorker(db, sender, renderer)
		go func() {
			log.Printf("Email worker: starting")
			if err := worker.Run(context.Background()); err != nil {
				log.Printf("Email worker stopped: %v", err)
			}
		}()

		verifyService = users.NewVerifyService(db, emailSvc, appBaseURL, cfg.Email.FromName)
		authH.SetVerifyService(verifyService)
		log.Printf("Email worker: verification service configured with base URL %s", appBaseURL)
	} else {
		log.Printf("Email worker: disabled via config")
	}

	// Account Verify Email route
	if verifyService != nil {
		accountVerifyEmailH := handlers.NewAccountVerifyEmailHandler(verifyService, flashCodec)
		authOnly.POST("/account/verify-email-send", accountVerifyEmailH.SendVerificationEmail)
	}

	// Password Change routes
	var passwordChangeSvc *users.PasswordChangeService
	if cfg.Email.Enabled && verifyService != nil {
		passwordChangeSvc = users.NewPasswordChangeService(db, emailSvc, appBaseURL, cfg.Email.FromName)
		accountH.SetPasswordChangeService(passwordChangeSvc)

		passwordConfirmH := handlers.NewPasswordChangeConfirmHandler(authRepo, passwordChangeSvc, flashCodec)
		r.GET("/confirm-password-change", passwordConfirmH.ConfirmPasswordChange)
		r.GET("/cancel-password-change", passwordConfirmH.CancelPasswordChange)
	}

	var shippingSvc *shipping.Service
	if cfg.Shipping.Enabled {
		var shipProvider shipping.Provider
		switch cfg.Shipping.Provider {
		case "", "mock":
			shipProvider = shipping.NewMockProvider(cfg.Shipping.MockBaseURL)
		default:
			log.Fatalf("unsupported shipping provider: %s", cfg.Shipping.Provider)
		}

		shippingSvc = shipping.NewService(db, shipProvider, emailSvc, appBaseURL)
		shipWorker := shipping.NewWorker(shippingSvc)
		go func() {
			log.Printf("Shipping worker: starting provider=%s", shipProvider.Name())
			if err := shipWorker.Run(context.Background()); err != nil {
				log.Printf("Shipping worker stopped: %v", err)
			}
		}()
	} else {
		log.Printf("Shipping worker: disabled via config")
	}

	// Note: FX worker is started in cmd/worker/main.go, not in the web server
	// The web server should not run background workers to avoid redundant processing

	// Admin Orders (depends on email/shipping services)
	refundSvc := payments.NewRefundService(db, provider, emailSvc, appBaseURL)
	adminSmsH := adminHandlers.NewSmsHandler(db, flashCodec, logger)
	admin.GET("/sms/failed", adminSmsH.ListFailed)

	adminOrders := adminHandlers.NewOrdersHandler(db, flashCodec, refundSvc, shippingSvc)
	admin.GET("/orders", adminOrders.List)
	admin.GET("/orders/:id", adminOrders.Detail)
	admin.GET("/orders/:id/refund", adminOrders.RefundForm)
	admin.POST("/orders/:id/refund", adminOrders.Refund)
	admin.POST("/orders/:id/shipments/label", adminOrders.CreateShipmentLabel)
	admin.POST("/orders/:id/shipments/manual", adminOrders.CreateManualShipment)
	admin.POST("/orders/:id/:action", adminOrders.Action) // ship|deliver|cancel|refund

	// Checkout & Orders
	orderSvc := orders.NewService(db, currencySvc)
	paySvc := payments.NewService(db, provider)
	checkoutH := handlers.NewCheckoutHandler(db, flashCodec, cartCK, cartSvc, orderSvc, emailSvc, currencySvc, appBaseURL)
	ordersH := handlers.NewOrdersHandler(db, flashCodec, paySvc)
	cartBadgeH := handlers.NewCartBadgeHandler(db)
	cartAddH := handlers.NewCartAddHandler(db)

	r.GET("/checkout", checkoutH.Get)
	r.POST("/checkout", checkoutH.Post)

	r.GET("/orders/:id", ordersH.Detail)
	r.GET("/orders/:id/invoice.pdf", ordersH.InvoicePDF)
	r.GET("/orders/:id/pay", ordersH.PayGet)
	r.POST("/orders/:id/pay", ordersH.PayPost)

	// HTMX cart endpoints
	r.GET("/api/cart/badge", cartBadgeH.GetBadge)
	r.POST("/api/cart/add", cartAddH.AddItem)

	// Webhooks (not CSRF-protected; signature is security layer)
	r.POST("/webhooks/mock", webhookH.Handle)

	return r
}

func mustSecret() []byte {
	// Prod’da APP_SECRET zorunlu olmalı.
	if s := os.Getenv("APP_SECRET"); len(s) >= 32 {
		return []byte(s)
	}
	// Dev fallback: process restart’te değişir; flash imzası için yeterli
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
