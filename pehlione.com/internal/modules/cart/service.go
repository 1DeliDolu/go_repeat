package cart

import (
	"context"
	"errors"
	"log"
	"math"
	"sort"
	"strings"

	"gorm.io/gorm"

	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/modules/currency"
	"pehlione.com/app/pkg/view"
)

type Service struct {
	db       *gorm.DB
	currency *currency.Service
}

func NewService(db *gorm.DB, curr *currency.Service) *Service {
	return &Service{db: db, currency: curr}
}

type cartRow struct {
	VariantID  string `gorm:"column:variant_id"`
	Qty        int    `gorm:"column:qty"`
	PriceCents int    `gorm:"column:price_cents"`
	Currency   string `gorm:"column:currency"`

	ProductName string `gorm:"column:product_name"`
	ProductSlug string `gorm:"column:product_slug"`
	ImageURL    string `gorm:"column:image_url"`
}

var ErrMixedCurrency = errors.New("cart contains multiple currencies")

func (s *Service) BuildCartPageForUser(ctx context.Context, userID string, displayCurrency string) (view.CartPage, error) {
	if userID == "" {
		return view.CartPage{}, errors.New("missing userID")
	}

	// First, find the user's open cart
	var cartID string
	err := s.db.WithContext(ctx).
		Model(&Cart{}).
		Where("user_id = ? AND status = ?", userID, "open").
		Order("updated_at DESC").
		Limit(1).
		Pluck("id", &cartID).Error
	if err != nil {
		return view.CartPage{}, err
	}
	log.Printf("BuildCartPageForUser: user=%s, cartID=%s", userID, cartID)

	const q = `
SELECT
  ci.variant_id AS variant_id,
  ci.quantity   AS qty,
  v.price_cents AS price_cents,
  v.currency    AS currency,
  p.name        AS product_name,
  p.slug        AS product_slug,
  '' AS image_url
FROM cart_items ci
JOIN product_variants v ON v.id = ci.variant_id
JOIN products p ON p.id = v.product_id
WHERE ci.cart_id = ?
ORDER BY ci.created_at ASC;
`

	var rows []cartRow
	if err := s.db.WithContext(ctx).Raw(q, cartID).Scan(&rows).Error; err != nil {
		return view.CartPage{}, err
	}
	log.Printf("BuildCartPageForUser: found %d items in cart %s", len(rows), cartID)

	return s.buildCartVMFromRows(ctx, rows, displayCurrency)
}

func (s *Service) BuildCartPageFromCookie(ctx context.Context, c *cartcookie.Cart, displayCurrency string) (view.CartPage, error) {
	if c == nil || len(c.Items) == 0 {
		return view.CartPage{Items: []view.CartItem{}, Currency: displayCurrency, BaseCurrency: s.baseCurrency()}, nil
	}

	// variantID -> qty
	qtyByID := make(map[string]int, len(c.Items))
	ids := make([]string, 0, len(c.Items))
	for _, it := range c.Items {
		if it.VariantID == "" || it.Qty <= 0 {
			continue
		}
		if _, ok := qtyByID[it.VariantID]; !ok {
			ids = append(ids, it.VariantID)
		}
		qtyByID[it.VariantID] += it.Qty
	}
	if len(ids) == 0 {
		return view.CartPage{Items: []view.CartItem{}}, nil
	}

	// IN sorgusu deterministik olsun diye sırala
	sort.Strings(ids)

	var rows []cartRow
	if err := s.db.WithContext(ctx).
		Table("product_variants AS v").
		Select(`v.id AS variant_id,
			0 AS qty,
			v.price_cents AS price_cents,
			v.currency AS currency,
			p.name AS product_name,
			p.slug AS product_slug,
			'' AS image_url`).
		Joins("JOIN products p ON p.id = v.product_id").
		Where("v.id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return view.CartPage{}, err
	}

	// DB sonuçlarını map'le
	infoByID := make(map[string]cartRow, len(rows))
	for _, r := range rows {
		infoByID[r.VariantID] = r
	}

	// Cookie sırasını koruyarak final rows üret
	final := make([]cartRow, 0, len(ids))
	for _, it := range c.Items {
		if it.VariantID == "" || it.Qty <= 0 {
			continue
		}
		r, ok := infoByID[it.VariantID]
		if !ok {
			// Variant silinmiş / yok: bu item'ı atlayalım
			continue
		}
		r.Qty = it.Qty
		final = append(final, r)
	}

	return s.buildCartVMFromRows(ctx, final, displayCurrency)
}

func (s *Service) buildCartVMFromRows(ctx context.Context, rows []cartRow, displayCurrency string) (view.CartPage, error) {
	vm := view.CartPage{Items: make([]view.CartItem, 0, len(rows))}

	itemCurrency := ""
	for _, r := range rows {
		if r.Currency != "" {
			itemCurrency = strings.ToUpper(strings.TrimSpace(r.Currency))
			break
		}
	}
	if itemCurrency == "" {
		itemCurrency = strings.ToUpper(strings.TrimSpace(s.baseCurrency()))
	}

	displayCurrency = strings.ToUpper(strings.TrimSpace(displayCurrency))
	rate := 1.0
	if s.currency != nil {
		base := strings.ToUpper(strings.TrimSpace(s.currency.BaseCurrency()))
		if itemCurrency != "" && base != "" && strings.EqualFold(itemCurrency, base) {
			displayCurrency = s.normalizeDisplayCurrency(ctx, displayCurrency)
			if rateInfo, err := s.currency.DisplayRate(ctx, displayCurrency); err == nil && rateInfo.Rate > 0 {
				rate = rateInfo.Rate
			}
		} else {
			// Avoid incorrect FX conversions when cart items are not in base currency.
			displayCurrency = itemCurrency
			rate = 1.0
		}
	}
	if displayCurrency == "" {
		displayCurrency = itemCurrency
	}

	subtotalBase := 0
	subtotalDisplay := 0
	count := 0

	for _, r := range rows {
		if r.Qty <= 0 {
			continue
		}
		if itemCurrency == "" {
			itemCurrency = strings.ToUpper(strings.TrimSpace(r.Currency))
		} else if r.Currency != "" && !strings.EqualFold(r.Currency, itemCurrency) {
			return view.CartPage{}, ErrMixedCurrency
		}

		line := r.PriceCents * r.Qty
		subtotalBase += line
		count += r.Qty

		convertedUnit := convertAmount(r.PriceCents, rate)
		convertedLine := convertAmount(line, rate)
		subtotalDisplay += convertedLine

		vm.Items = append(vm.Items, view.CartItem{
			ProductName: r.ProductName,
			ProductSlug: r.ProductSlug,
			ImageURL:    r.ImageURL,
			VariantID:   r.VariantID,
			Qty:         r.Qty,

			UnitPriceCents:     convertedUnit,
			LineTotalCents:     convertedLine,
			BaseUnitPriceCents: r.PriceCents,
			BaseLineTotalCents: line,

			UnitPrice: view.MoneyFromCents(convertedUnit, displayCurrency),
			LineTotal: view.MoneyFromCents(convertedLine, displayCurrency),
		})
	}

	vm.Currency = displayCurrency
	vm.BaseCurrency = itemCurrency
	vm.Count = count
	vm.SubtotalCents = subtotalBase
	vm.DisplaySubtotalCents = subtotalDisplay
	vm.Subtotal = view.MoneyFromCents(subtotalDisplay, displayCurrency)
	vm.BaseSubtotalCents = subtotalBase

	vm.TotalCents = subtotalBase
	vm.DisplayTotalCents = subtotalDisplay
	vm.Total = view.MoneyFromCents(subtotalDisplay, displayCurrency)
	vm.BaseTotalCents = subtotalBase

	return vm, nil
}

func (s *Service) baseCurrency() string {
	if s.currency != nil {
		return s.currency.BaseCurrency()
	}
	return ""
}

func (s *Service) normalizeDisplayCurrency(_ context.Context, current string) string {
	if s.currency == nil {
		if current == "" {
			return s.baseCurrency()
		}
		return current
	}
	if normalized, ok := s.currency.NormalizeDisplay(current); ok {
		return normalized
	}
	return s.currency.DefaultDisplayCurrency()
}

func convertAmount(base int, rate float64) int {
	if rate == 1 {
		return base
	}
	val := float64(base) * rate
	if val >= 0 {
		return int(math.Round(val))
	}
	return -int(math.Round(math.Abs(val)))
}
