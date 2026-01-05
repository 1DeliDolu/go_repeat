
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

### Teknik Özellikler
- Server-Side Rendering (Templ)
- Type-safe templates
- Flash message sistemi
- Error handling middleware
- Request ID tracking
- Structured logging (slog)

## 🏗️ Proje Yapısı

---
```
pehlione.com/
├── cmd/
│   └── web/           # Ana uygulama entry point
├── internal/
│   ├── http/
│   │   ├── handlers/  # HTTP request handlers
│   │   │   ├── admin/ # Admin panel handlers
│   │   │   ├── cart.go
│   │   │   ├── checkout.go
│   │   │   ├── orders.go
│   │   │   └── auth.go
│   │   ├── middleware/ # Middleware katmanı
│   │   │   ├── auth.go
│   │   │   ├── csrf.go
│   │   │   ├── cart_badge.go
│   │   │   └── flash.go
│   │   ├── cartcookie/ # Cookie-based cart codec
│   │   ├── flash/      # Flash message codec
│   │   └── router.go   # Route tanımları
│   ├── modules/
│   │   ├── cart/       # Sepet business logic
│   │   │   ├── models.go
│   │   │   ├── repo.go
│   │   │   └── service.go
│   │   ├── orders/     # Sipariş business logic
│   │   │   ├── models.go
│   │   │   ├── repo.go
│   │   │   ├── service.go
│   │   │   └── errors.go
│   │   ├── checkout/   # Checkout logic
│   │   └── payments/   # Ödeme entegrasyonu
│   └── shared/
│       └── apperr/     # Application errors
├── pkg/
│   └── view/           # View models
│       ├── cart.go
│       ├── checkout.go
│       └── flash.go
├── templates/
│   ├── layout/         # Layout bileşenleri
│   │   ├── base.templ
│   │   └── header.templ
│   └── pages/          # Sayfa templates
│       ├── cart.templ
│       ├── checkout.templ
│       ├── products.templ
│       └── home.templ
├── static/             # Static assets
├── migrations/         # Database migrations (goose)
└── magefile.go         # Build automation (Mage)
```
---

## 🗄️ Database Schema

### Core Tables
- **users** - Kullanıcı bilgileri (id, email, password_hash, role)
- **sessions** - Oturum yönetimi
- **carts** - Alışveriş sepetleri (id, user_id, status)
- **cart_items** - Sepet içerikleri (cart_id, variant_id, quantity)

### Product Tables
- **products** - Ürün bilgileri (id, name, slug, status)
- **product_variants** - Ürün varyantları (id, product_id, sku, price_cents, stock)

### Order Tables
- **orders** - Siparişler (id, user_id, guest_email, status, total_cents)
- **order_items** - Sipariş kalemleri
- **order_events** - Sipariş durum geçişleri

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
```bash
git clone <repo-url>
cd pehlione.com
```
---

2. **Bağımlılıkları yükleyin**
---
```bash
go mod download
npm install  # Tailwind için
```
---

3. **Environment variables ayarlayın**
---
```bash
# .env dosyası oluşturun
cp .env.example .env
```
---

Gerekli değişkenler:
---
```env
DB_DSN=user:pass@tcp(localhost:3306)/pehlione_go?parseTime=true
SECRET_KEY=<64-char-hex-secret>
SESSION_TTL_HOURS=168
```
---

4. **Database migration**
---
```bash
goose -dir migrations mysql "user:pass@/pehlione_go" up
```
---

5. **Templ generate**
---
```bash
templ generate
```
---

6. **Build ve çalıştır**
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
```
GET  /account/orders      # Kullanıcı siparişleri
GET  /orders/:id          # Sipariş detayı
POST /orders/:id/pay      # Ödeme başlat (CSRF)
```
---

### Admin Routes
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

### 1. Cart Validation
- En az 1 ürün kontrolü
- Currency consistency check

### 2. Form Submission
- CSRF token validation
- Address validation (go-playground/validator)
- Email validation (guest için zorunlu)

### 3. Order Creation (Transaction)
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

### Type-safe Components
---
```go
templ Cart(flash *view.Flash, p view.CartPage) {
    @layout.Base("Shopping Cart", flash, CartBody(p))
}
```
---

### View Models
- **view.CartPage** - Sepet görünümü için
- **view.CheckoutForm** - Checkout form data
- **view.CheckoutSummary** - Sipariş özeti
- **view.HeaderCtx** - Header context (auth, cart badge)

### Template Generation
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
```go
log.Printf("CartAdd: error adding item: %v", err)
log.Printf("Checkout error (unhandled): %T - %v", err, err)
```
---

### Request Tracking
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

### In Progress
- [ ] Product images (product_images table missing)
- [ ] Order payment integration
- [ ] Order refund flow
- [ ] Email notifications

### Future Enhancements
- [ ] Product search & filtering
- [ ] Wishlist functionality
- [ ] Customer reviews
- [ ] Multi-currency support
- [ ] Shipping integrations
- [ ] Invoice generation

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open Pull Request

## 📄 License

MIT License - see LICENSE file for details

## 📞 Contact

Project Link: [https://github.com/1DeliDolu/go_repeat](https://github.com/1DeliDolu/go_repeat)
