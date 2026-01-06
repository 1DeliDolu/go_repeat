package http

import (
	"context"
	"crypto/rand"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/handlers"
	adminHandlers "pehlione.com/app/internal/http/handlers/admin"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/internal/modules/email"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/payments"
	"pehlione.com/app/internal/modules/products"
	"pehlione.com/app/internal/modules/users"
	"pehlione.com/app/internal/storage"
	"pehlione.com/app/pkg/view"
)

func NewRouter(logger *slog.Logger, db *gorm.DB) *gin.Engine {
	// --- Secrets / Codecs ---
	secret := mustSecret()
	appBaseURL := envOr("APP_BASE_URL", "http://localhost:8080")

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

	// Cart cookie
	cartCKName := envOr("CART_COOKIE_NAME", "pehlione_cart")
	cartCKSecure := envBool("CART_COOKIE_SECURE", false)
	cartCK := cartcookie.New(secret, cartCKName, cartCKSecure)
	cartSvc := cart.NewService(db)

	// --- Router + Middleware order ---
	r := gin.New()

	// Core observability first
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	// Request-scoped UI context
	r.Use(middleware.FlashMiddleware(flashCodec))
	r.Use(middleware.CSRF(csrfCfg))
	r.Use(middleware.SessionMiddleware(sessCfg))

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
	productsH := handlers.NewProductsHandler(productsSvc)
	r.GET("/products", productsH.List)
	r.GET("/products/:slug", productsH.Show)

	// Cart codec
	cartCodec := cartcookie.New(secret, "pehlione_cart", false) // dev: secure=false

	// Cart (public shopping cart page)
	cartH := handlers.NewCartHandler(db, flashCodec, cartCodec)
	r.GET("/cart", cartH.Get)
	r.POST("/cart/items", cartH.Add) // SSR: add to cart + redirect (form submission dari product pages)

	// Company (public company info page)
	companyH := handlers.NewCompanyHandler()
	r.GET("/company", companyH.Get)

	// Demo: flash set + redirect (POST => CSRF token gerekir)
	r.POST("/demo/flash", func(c *gin.Context) {
		render.RedirectWithFlash(c, flashCodec, "/", view.FlashSuccess, "İşlem başarılı (flash).")
	})

	// Auth (DB-backed): signup/login/logout
	authH := handlers.NewAuthHandlers(db, flashCodec, sessCfg, cartCK)

	r.GET("/signup", authH.SignupGet)
	r.POST("/signup", authH.SignupPost)
	r.GET("/verify", authH.VerifyGet)
	r.POST("/verify", authH.VerifyPost)
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
	mockSecret := envOr("MOCK_WEBHOOK_SECRET", "dev_secret_change_me")
	provider := payments.NewMockProvider(mockSecret)

	// Webhook service + handler
	webhookSvc := payments.NewWebhookService(db)
	webhookH := handlers.NewWebhookHandler(logger, provider, webhookSvc)

	// Admin Orders
	refundSvc := payments.NewRefundService(db, provider)
	adminOrders := adminHandlers.NewOrdersHandler(db, refundSvc)
	admin.GET("/orders", adminOrders.List)
	admin.GET("/orders/:id", adminOrders.Detail)
	admin.POST("/orders/:id/:action", adminOrders.Action) // ship|deliver|cancel|refund
	admin.POST("/products/:id/variants/:vid/delete", ph.DeleteVariant)
	admin.POST("/products/:id/variants/:vid", ph.UpdateVariant)
	admin.POST("/products/:id/variants/:vid/sku", ph.UpdateVariantSKU)

	admin.POST("/products/:id/images", ph.AddImage)
	admin.POST("/products/:id/images/:iid/delete", ph.DeleteImage)
	admin.POST("/products/:id/images/upload", ph.UploadImage)

	// Protected routes (require authentication)
	authOnly := r.Group("")
	authOnly.Use(middleware.RequireAuth(flashCodec))
	authOnly.GET("/account", handlers.Account)

	// Account routes
	ordersRepo := orders.NewRepo(db)
	accountH := handlers.NewAccountOrdersHandler(ordersRepo)
	account := r.Group("/account")
	account.Use(middleware.RequireAuth(flashCodec))
	account.GET("/orders", accountH.List)

	// --- Email worker initialization ---
	if envBool("EMAIL_SEND_ENABLED", true) {
		emailSvc = email.NewService(db)

		// Create SMTP sender
		smtpCfg := email.SMTPCfg{
			Host:   envOr("SMTP_HOST", "localhost"),
			Port:   envInt("SMTP_PORT", 1025),
			User:   os.Getenv("SMTP_USER"),
			Pass:   os.Getenv("SMTP_PASS"),
			From:   envOr("SMTP_FROM", "no-reply@pehlione.com"),
			UseTLS: envBool("SMTP_USE_TLS", false),
		}
		log.Printf("Email worker: initializing with SMTP host=%s port=%d from=%s", smtpCfg.Host, smtpCfg.Port, smtpCfg.From)
		sender := email.NewSMTPSender(smtpCfg)

		// Start worker in background
		worker := email.NewWorker(db, sender)
		go func() {
			log.Printf("Email worker: starting")
			if err := worker.Run(context.Background()); err != nil {
				log.Printf("Email worker stopped: %v", err)
			}
		}()

		// Inject email service and verify service into auth handlers
		verifyService := users.NewVerifyService(db, emailSvc, appBaseURL)
		authH.SetEmailService(emailSvc)
		authH.SetVerifyService(verifyService)
		log.Printf("Email worker: verification service configured with base URL %s", appBaseURL)
	} else {
		log.Printf("Email worker: disabled (EMAIL_SEND_ENABLED=false)")
	}

	// Checkout & Orders
	orderSvc := orders.NewService(db)
	paySvc := payments.NewService(db, provider)
	checkoutH := handlers.NewCheckoutHandler(db, flashCodec, cartCK, orderSvc, emailSvc, appBaseURL)
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
