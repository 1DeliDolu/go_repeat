
# Pehlione E-Commerce Platform

Go tabanlı, server-side rendering (SSR) kullanan modern e-ticaret platformu.

## 🚀 Özellikler

### Kullanıcı Özellikleri
- ✅ Kullanıcı kaydı ve kimlik doğrulama (session-based)
- ✅ Guest (misafir) ve kayıtlı kullanıcı desteği
- ✅ Role-based yetkilendirme (user/admin)
- ✅ CSRF koruması (double-submit cookie pattern)

### Alışveriş Sepeti
- ✅ DB tabanlı sepet (kayıtlı kullanıcılar)
- ✅ Cookie tabanlı sepet (misafir kullanıcılar)
- ✅ Gerçek zamanlı sepet badge güncellemesi
- ✅ Session cache optimizasyonu
- ✅ Aynı ürünün miktar artırma desteği

### Ödeme ve Sipariş
- ✅ Checkout akışı (adres formu, kargo seçimi)
- ✅ Guest checkout (email ile)
- ✅ Kayıtlı kullanıcı checkout
- ✅ Idempotency key desteği (tekrar sipariş önleme)
- ✅ Stok kontrolü ve rezervasyon
- ✅ Sipariş detay sayfası
- ✅ Admin sipariş yönetimi

### Technical Features
- Server-Side Rendering (Templ)
- Type-safe templates with component architecture
- Reusable product card components (StandardProductCard, SaleProductCard)
- Accessibility features (ARIA labels, SR-only headings, dialog roles)
- Performance optimizations (lazy-loading images, async decoding)
- Async email system with outbox pattern
- Payment provider abstraction
- PDF invoice generation
- Refund processing with webhooks
- Flash message system
- Error handling middleware
- Request ID tracking
- Structured logging (slog)
- CSRF protection (double-submit cookie pattern)

## 🏗️ Proje Yapısı

---
```
pehlione.com/
├── cmd/
│   └── web/           # Main application entry point
├── internal/
│   ├── http/
│   │   ├── handlers/  # HTTP request handlers
│   │   │   ├── admin/ # Admin panel handlers
│   │   │   ├── cart.go
│   │   │   ├── checkout.go
│   │   │   ├── orders.go
│   │   │   ├── products.go
│   │   │   └── auth.go
│   │   ├── middleware/ # Middleware layer
│   │   │   ├── auth.go
│   │   │   ├── csrf.go
│   │   │   ├── cart_badge.go
│   │   │   └── flash.go
│   │   ├── cartcookie/ # Cookie-based cart codec
│   │   ├── flash/      # Flash message codec
│   │   └── router.go   # Route definitions
│   ├── modules/
│   │   ├── auth/       # Authentication logic
│   │   ├── cart/       # Cart business logic
│   │   ├── checkout/   # Checkout logic
│   │   ├── email/      # Email outbox service (async)
│   │   │   ├── models.go
│   │   │   ├── service.go
│   │   │   ├── worker.go
│   │   │   ├── smtp_sender.go
│   │   │   └── mailtrap.go
│   │   ├── orders/     # Order business logic
│   │   │   ├── models.go
│   │   │   ├── repo.go
│   │   │   ├── service.go
│   │   │   ├── admin_service.go
│   │   │   └── errors.go
│   │   ├── payments/   # Payment integration
│   │   │   ├── provider.go
│   │   │   ├── provider_mock.go
│   │   │   ├── service.go
│   │   │   ├── refund_service.go
│   │   │   └── webhook_service.go
│   │   ├── products/   # Product management
│   │   └── users/      # User management
│   ├── pdf/           # PDF invoice generation
│   │   └── invoice.go
│   └── shared/
│       └── apperr/     # Application errors
├── pkg/
│   └── view/           # View models
│       ├── cart.go
│       ├── checkout.go
│       └── flash.go
├── templates/
│   ├── components/     # Reusable UI components
│   ├── layout/         # Layout components
│   │   └── base.templ
│   ├── shared/         # Shared template utilities
│   │   ├── base.templ
│   │   └── money.go
│   └── pages/          # Page templates
│       ├── products/
│       │   ├── index.templ  # Product listing with StandardProductCard/SaleProductCard
│       │   └── show.templ   # Product detail page
│       ├── cart.templ
│       ├── checkout.templ
│       ├── order_detail.templ
│       ├── order_pay.templ
│       ├── account_orders.templ
│       ├── admin_*.templ    # Admin panel pages
│       └── home.templ
├── static/             # Static assets (CSS, JS, images)
├── storage/            # File storage (product images)
├── migrations/         # Database migrations (goose)
└── magefile.go         # Build automation (Mage)
```
---
## 🗄️ Database Schema (Extended)

### Core Tables
- **users** - Kullanıcı bilgileri (id, email, password_hash, role)
- **sessions** - Oturum yönetimi
- **carts** - Alışveriş sepetleri (id, user_id, status)
- **cart_items** - Sepet içerikleri (cart_id, variant_id, quantity)

### Product Tables
- **products** - Product information (id, name, slug, status)
- **product_variants** - Product variants (id, product_id, sku, price_cents, stock)
- **product_images** - Product images (id, product_id, storage_key, url, display_order)

### Order Tables
- **orders** - Orders (id, user_id, guest_email, status, total_cents)
- **order_items** - Order line items
- **order_events** - Order status transitions

### Email & Payment Tables
- **outbox_emails** - Async email queue (id, to_email, subject, body_html, status, attempts)
- **payment_intents** - Payment tracking (id, order_id, provider, status, amount_cents)
- **refunds** - Refund records (id, payment_intent_id, amount_cents, status)

## 🛠️ Teknoloji Stack

### Backend
- **Go 1.22+** - Programming language
- **Gin** - HTTP web framework
- **GORM** - ORM (MySQL)
- **Templ** - Type-safe Go templates

### Frontend
- **Tailwind CSS** - Utility-first CSS
- **Vanilla JavaScript** - Client-side interactions
- Server-Side Rendering (no SPA)

### Tools
- **Mage** - Build automation
- **Air** - Hot reload development
- **Goose** - Database migrations

## 📦 Kurulum

### Gereksinimler
- Go 1.22 veya üzeri
- MySQL 8.0+
- Node.js (Tailwind CSS için)

### Adımlar

1. **Projeyi klonlayın**
---
---
---
```bash
git clone <repo-url>
cd pehlione.com
```
---
2. **Bağımlılıkları yükleyin**
---
---
---
```bash
go mod download
npm install  # Tailwind için
```
---
3. **Environment variables ayarlayın**
---
---
---
```bash
# .env dosyası oluşturun
cp .env.example .env
```
---
Gerekli değişkenler:
---
---
---
```env
DB_DSN=user:pass@tcp(localhost:3306)/pehlione_go?parseTime=true
SECRET_KEY=<64-char-hex-secret>
SESSION_TTL_HOURS=168
```
---
4. **Database migration**
---
---
---
```bash
goose -dir migrations mysql "user:pass@/pehlione_go" up
```
---
5. **Templ generate**
---
---
---
```bash
templ generate
```
---
6. **Build ve çalıştır**
---
---
---
```bash
# Development (hot reload)
mage dev

# Production build
mage build
./bin/pehlione-web.exe
```
---
## 🔐 Güvenlik

### Implemented Security Features
- ✅ CSRF Protection (double-submit cookie)
- ✅ Password hashing (bcrypt)
- ✅ Session management with secure cookies
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS protection (template auto-escaping)
- ✅ Input validation (go-playground/validator)

### Cookie Settings
- `SameSite=Lax` - CSRF koruması
- `HttpOnly=true` - Session cookies için XSS önleme
- `Secure=true` - Production'da HTTPS zorunluluğu

## 🧪 Test Kullanıcıları

Database seed migration ile oluşturulur:

| Email | Password | Role |
|-------|----------|------|
| delione@pehlione.com | password123 | admin |
| deli@pehlione.com | password123 | user |

## 📝 API Endpoints

### Public Routes
---
---
---
```
GET  /                    # Ana sayfa
GET  /products            # Ürün listesi
GET  /cart                # Sepet sayfası
POST /cart/add            # Sepete ekleme (CSRF)
GET  /checkout            # Checkout sayfası
POST /checkout            # Sipariş oluşturma (CSRF)
GET  /signup              # Kayıt formu
POST /signup              # Kayıt işlemi (CSRF)
GET  /login               # Giriş formu
POST /login               # Giriş işlemi (CSRF)
POST /logout              # Çıkış (CSRF)
```
---
### Authenticated Routes
---
---
---
```
GET  /account/orders      # Kullanıcı siparişleri
GET  /orders/:id          # Sipariş detayı
POST /orders/:id/pay      # Ödeme başlat (CSRF)
```
---
### Admin Routes
---
---
---
```
GET  /admin/orders        # Tüm siparişler
GET  /admin/orders/:id    # Sipariş detayı
POST /admin/orders/:id    # Sipariş aksiyonu (CSRF)
```
---
## 🚦 Middleware Stack

Request işleme sırası:
1. **RequestID** - Her request için unique ID
2. **Logger** - Structured logging (slog)
3. **Flash** - Flash message handling
4. **CSRF** - CSRF token validation
5. **Session** - Session management
6. **CartBadge** - Cart count için DB/cookie query
7. **ErrorHandler** - Structured error handling
8. **Recovery** - Panic recovery

## 🔄 Cart Flow

### Guest User (Cookie-based)
1. User adds product → POST /cart/add
2. Handler reads cookie cart or creates new
3. Item added to cookie (base64 JSON)
4. Redirect to /cart with flash message
5. Cart page reads from cookie

### Logged-in User (DB-based)
1. User adds product → POST /cart/add
2. Handler gets or creates cart (DB)
3. Item added to cart_items table
4. Session cache cleared
5. Redirect to /cart with flash message
6. Cart page reads from DB with JOIN

### Guest → Logged-in Migration
- Login sonrası cookie cart otomatik DB cart'a merge edilir
- Cookie temizlenir

## 💳 Checkout Flow

## 📧 Email System (Outbox Pattern)

### Architecture
- **Outbox Table** - Reliable email delivery with retry logic
- **Background Worker** - Processes pending emails asynchronously
- **Multiple Senders** - SMTP, Mailtrap (test mode)
- **Retry Strategy** - Exponential backoff for failed sends

### Email Flow
---

---
```go
// 1. Enqueue email (in transaction with order creation)
emailSvc.Enqueue(ctx, order.Email, "Order Confirmation", text, html)

// 2. Background worker polls outbox
emails := emailSvc.GetPending(ctx, 10)

// 3. Send via configured provider
for _, email := range emails {
    err := sender.Send(ctx, Message{
        To: email.ToEmail,
        Subject: email.Subject,
        HTML: *email.BodyHTML,
    })
    // Update status (sent/failed) with retry logic
}
```
---
## 💳 Payment & Refund System

### Payment Provider Interface
- **Provider interface** - Abstraction for payment gateways
- **Mock provider** - Development/testing implementation
- **Payment intents** - Track payment lifecycle
- **Webhook handling** - Process provider callbacks

### Refund Service
- **Full and partial refunds** - Flexible refund amounts
- **Webhook integration** - Automatic refund processing
- **Status tracking** - Refund lifecycle management
- **Database persistence** - Refund records and history

## 📄 PDF Invoice Generation

### Features
- **Branded invoices** - Company logo and colors (pehliONE yellow/orange)
- **Order details** - Line items, quantities, prices
- **Totals breakdown** - Subtotal, shipping, tax, total
- **Customer info** - Billing address and contact details
- **go-pdf/fpdf** - Native Go PDF generation (no external dependencies)

### 1. Cart Validation
- En az 1 ürün kontrolü
- Currency consistency check

### 2. Form Submission
- CSRF token validation
- Address validation (go-playground/validator)
- Email validation (guest için zorunlu)

### 3. Order Creation (Transaction)
---
---
---
```
1. Read cart items
2. Lock product variants (FOR UPDATE)
3. Validate stock availability
4. Deduct stock
5. Calculate totals
6. Create order record
7. Create order_items
8. Clear cart (DB or cookie)
```
---
### 4. Stock Management
- Pessimistic locking (SELECT FOR UPDATE)
- Atomic stock deduction
- OutOfStockError handling

## 🎨 Template System (Templ)

## ♿ Accessibility & Performance

### Accessibility Features
- ✅ ARIA labels and landmarks (`aria-labelledby`, `aria-modal`)
- ✅ SR-only headings for screen readers
- ✅ Proper dialog roles with labeled headings
- ✅ Semantic HTML structure
- ✅ Keyboard navigation support
- ✅ Color contrast compliance

### Performance Optimizations
- ✅ Lazy-loading images (`loading="lazy"`)
- ✅ Async image decoding (`decoding="async"`)
- ✅ Session cache for cart badge
- ✅ Component-based templates (reduced duplication)
- ✅ Optimized database queries with eager loading

### Component Architecture
Product pages use reusable template components to ensure consistency and maintainability:

**StandardProductCard**
- Standard product display with hover effects
- Disabled state for out-of-stock items
- Lazy-loaded images
- Add to cart form with CSRF protection

**SaleProductCard**
- Sale badge overlay
- Rose-themed styling for discounted items
- Same structure as StandardProductCard with visual emphasis

Benefits:
- Single source of truth for product card markup
- Consistent behavior across the application
- Easier maintenance and updates
- Type-safe props with Go templating

### Type-safe Components
---
---
---
```go
// Reusable product card components
templ StandardProductCard(p ProductCardVM, csrf string) {
    <div class="group flex flex-col rounded-xl border border-gray-100 bg-white p-4...">
        <a href={ fmt.Sprintf("/products/%s", p.Slug) }>
            if p.ImageURL != "" {
                <img src={ p.ImageURL } loading="lazy" decoding="async" .../>
            }
        </a>
        // ... button with out-of-stock handling
    </div>
}

templ SaleProductCard(p ProductCardVM, csrf string) {
    // Similar structure with sale-specific styling
}

// Page template using components
templ ProductsIndexPage(vm ProductsIndexVM) {
    @shared.Base(shared.BaseVM{Title: vm.Title}) {
        <section aria-labelledby="products-heading">
            <h2 id="products-heading" class="sr-only">Products</h2>
            for _, p := range vm.SaleProducts {
                @SaleProductCard(p, vm.CSRFToken)
            }
        </section>
    }
}
```
---
### View Models
- **view.CartPage** - Cart view with items
- **view.CheckoutForm** - Checkout form data
- **view.CheckoutSummary** - Order summary
- **view.HeaderCtx** - Header context (auth, cart badge)
- **ProductsIndexVM** - Product listing page (with CategoryGroups, SaleProducts)
- **ProductCardVM** - Individual product card data
- **ProductDetailVM** - Product detail page with variants

### Template Generation
---
---
---
```bash
# Generate _templ.go files
templ generate

# Watch mode (development)
templ generate --watch
```
---
## 📊 Monitoring & Logging

### Structured Logging
---
---
---
```go
log.Printf("CartAdd: error adding item: %v", err)
log.Printf("Checkout error (unhandled): %T - %v", err, err)
```
---
### Request Tracking
---
---
---
```json
{
  "time":"2026-01-05T18:37:30Z",
  "level":"WARN",
  "msg":"http_request",
  "request_id":"985f311591c8a69d",
  "method":"POST",
  "path":"/checkout",
  "status":400,
  "latency":13270700,
  "client_ip":"::1"
}
```
---
## 🐛 Known Issues & TODOs

### Recent Improvements ✅
- [x] Component-based product cards (StandardProductCard, SaleProductCard)
- [x] Accessibility enhancements (ARIA labels, SR-only headings, dialog roles)
- [x] Image performance optimization (lazy-loading, async decoding)
- [x] Out-of-stock handling in product cards
- [x] English UI translations
- [x] Product images table and storage system
- [x] Email notification system (outbox pattern with worker)
- [x] PDF invoice generation
- [x] Payment integration (with mock provider)
- [x] Refund service and webhook handling

### In Progress / Needs Migration
- [ ] Refund fields in orders table (RefundedCents, RefundedAt - currently in Go struct only)
- [ ] Email worker deployment configuration
- [ ] Payment provider production credentials

### Future Enhancements
- [ ] Product search & filtering
- [ ] Wishlist functionality
- [ ] Customer reviews
- [ ] Multi-currency support
- [ ] Real payment provider integration (Stripe, PayPal)
- [ ] Shipping provider integrations
- [ ] Advanced email templates
- [ ] SMS notifications

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open Pull Request

## 📄 License

MIT License - see LICENSE file for details

## 📞 Contact

Project Link: [https://github.com/1DeliDolu/pehlione_go](https://github.com/1DeliDolu/pehlione_go)
