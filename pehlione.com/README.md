
# Pehlione E-Commerce Platform

Modern e-commerce platform built with Go, using server-side rendering (SSR).

## 🚀 Features

### User Features
- ✅ User registration and authentication (session-based)
- ✅ Guest and registered user support
- ✅ Role-based authorization (user/admin)
- ✅ CSRF protection (double-submit cookie pattern)

### Shopping Cart
- ✅ Database-based cart (registered users)
- ✅ Cookie-based cart (guest users)
- ✅ Real-time cart badge updates
- ✅ Session cache optimization
- ✅ Quantity increment for same products

### Payment & Orders
- ✅ Checkout flow (address form, shipping selection)
- ✅ Guest checkout (email-based)
- ✅ Registered user checkout
- ✅ Idempotency key support (prevent duplicate orders)
- ✅ Stock control and reservation
- ✅ Order detail page
- ✅ Admin order management

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

## 🏗️ Project Structure

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
## 🗄️ Database Schema

### Core Tables
- **users** - User information (id, email, password_hash, role)
- **sessions** - Session management
- **carts** - Shopping carts (id, user_id, status)
- **cart_items** - Cart contents (cart_id, variant_id, quantity)

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

## 🛠️ Technology Stack

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

## 📦 Installation

### Requirements
- Go 1.22 or higher
- MySQL 8.0+
- Node.js (for Tailwind CSS)

### Steps

1. **Clone the repository**
---
```bash
git clone <repo-url>
cd pehlione.com
```
---
2. **Install dependencies**
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
3. **Set up environment variables**
```
---
# Create .env file
cp .env.example .env
---
```env
Required variables:
```
---
4. **Database migration**
---
```bash
goose -dir migrations mysql "user:pass@/pehlione_go" up
```
---
5. **Generate Templ**
---
```bash
templ generate
```
---
6. **Build and run**
---
```bash
# Development (hot reload)
mage dev

# Production build
mage build
./bin/pehlione-web.exe
```
---
## 🔐 Security

### Implemented Security Features
- ✅ CSRF Protection (double-submit cookie)
- ✅ Password hashing (bcrypt)
- ✅ Session management with secure cookies
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS protection (template auto-escaping)
- ✅ Input validation (go-playground/validator)

### Cookie Settings
- `SameSite=Lax` - CSRF protection
- `HttpOnly=true` - XSS prevention for session cookies
- `Secure=true` - HTTPS enforcement in production

## 🧪 Test Users

Created via database seed migration:

| Email | Password | Role |
|-------|----------|------|
| delione@pehlione.com | password123 | admin |
| deli@pehlione.com | password123 | user |

## 📝 API Endpoints

### Public Routes
---
```
GET  /                    # Home page
GET  /products            # Product listing
GET  /cart                # Cart page
POST /cart/add            # Add to cart (CSRF)
GET  /checkout            # Checkout page
POST /checkout            # Create order (CSRF)
GET  /signup              # Signup form
POST /signup              # Signup process (CSRF)
GET  /login               # Login form
POST /login               # Login process (CSRF)
POST /logout              # Logout (CSRF)
```
---
### Authenticated Routes
---
```
GET  /account/orders      # User orders
GET  /orders/:id          # Order detail
POST /orders/:id/pay      # Start payment (CSRF)
```
---
### Admin Routes
---
```
GET  /admin/orders        # All orders
GET  /admin/orders/:id    # Order detail
POST /admin/orders/:id    # Order action (CSRF)
```
---
## 🚦 Middleware Stack

Request processing order:
1. **RequestID** - Unique ID for each request
2. **Logger** - Structured logging (slog)
3. **Flash** - Flash message handling
4. **CSRF** - CSRF token validation
5. **Session** - Session management
6. **CartBadge** - Cart count DB/cookie query
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
- Cookie cart automatically merged to DB cart after login
- Cookie is cleared

## 💳 Checkout Flow

## 📧 Email System (Outbox Pattern)

### Architecture
- **Outbox Table** - Reliable email delivery with retry logic
- **Background Worker** - Processes pending emails asynchronously
- **Multiple Senders** - SMTP, Mailtrap (test mode)
- **Retry Strategy** - Exponential backoff for failed sends

### Email Flow

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
- Minimum 1 product check
- Currency consistency check

### 2. Form Submission
- CSRF token validation
- Address validation (go-playground/validator)
- Email validation (required for guests)

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
- [ ] Real payment provider integration (Stripe, PayPal)
- [ ] Advanced email templates with dynamic content
- [ ] Shipping carrier integrations (FedEx, UPS, DHL tracking)

## 🔀 GitHub Setup & Branch Management

### Initial Repository Setup

---
```bash
# Initialize git repository (if not already done)
git init

# Add remote repository
git remote add origin https://github.com/1DeliDolu/go_repeat.git

# Check remote configuration
git remote -v

# Push to main branch
git add .
git commit -m "Initial commit"
git branch -M main
git push -u origin main
```
---

### Branch Management

---
```bash
# Create and switch to a new branch
git checkout -b feature/new-feature

# Or create branch without switching
git branch feature/new-feature

# List all branches
git branch -a

# Switch between branches
git checkout main
git checkout feature/new-feature

# Push new branch to remote
git push -u origin feature/new-feature

# Delete a local branch
git branch -d feature/old-feature

# Delete a remote branch
git push origin --delete feature/old-feature
```
---

### Common Branch Workflows

**Feature Development Workflow:**

---
```bash
# 1. Create feature branch from main
git checkout main
git pull origin main
git checkout -b feature/shopping-cart-improvements

# 2. Make changes and commit regularly
git add .
git commit -m "Add cart quantity update feature"

# 3. Push to remote
git push -u origin feature/shopping-cart-improvements

# 4. Keep feature branch updated with main
git checkout main
git pull origin main
git checkout feature/shopping-cart-improvements
git merge main

# 5. When ready, create Pull Request on GitHub
```
---

**Hotfix Workflow:**

---
```bash
# 1. Create hotfix branch from main
git checkout main
git checkout -b hotfix/critical-bug-fix

# 2. Fix the issue and commit
git add .
git commit -m "Fix critical payment processing bug"

# 3. Push and create PR
git push -u origin hotfix/critical-bug-fix

# 4. After merge, pull latest main
git checkout main
git pull origin main
```
---

### Recommended Branch Naming Conventions

- `feature/` - New features (e.g., `feature/user-authentication`)
- `bugfix/` - Bug fixes (e.g., `bugfix/cart-calculation-error`)
- `hotfix/` - Critical production fixes (e.g., `hotfix/payment-gateway-issue`)
- `refactor/` - Code refactoring (e.g., `refactor/checkout-service`)
- `docs/` - Documentation updates (e.g., `docs/api-documentation`)
- `test/` - Adding or updating tests (e.g., `test/cart-unit-tests`)
- `chore/` - Maintenance tasks (e.g., `chore/update-dependencies`)

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
