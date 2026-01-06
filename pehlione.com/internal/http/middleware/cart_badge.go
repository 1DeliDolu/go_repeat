package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/cartcookie"
	cartmod "pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/pkg/view"
)

const cartBadgeKey = "cart_badge_qty"
const cartPreviewKey = "cart_preview"

// Cookie qty hesaplamak için "hook".
type CookieQtyFunc func(c *gin.Context) (int, error)

type CartBadgeCfg struct {
	DB         *gorm.DB
	CookieQty  CookieQtyFunc
	CartSvc    *cartmod.Service
	CartCookie *cartcookie.Codec
}

func CartBadge(cfg CartBadgeCfg) gin.HandlerFunc {
	return func(c *gin.Context) {
		qty := 0
		preview := view.CartPage{}
		previewSet := false

		if cfg.CartSvc != nil {
			if u, ok := CurrentUser(c); ok {
				if page, err := cfg.CartSvc.BuildCartPageForUser(c, u.ID); err == nil {
					preview = page
					previewSet = true
					qty = page.Count
				}
			} else if cfg.CartCookie != nil {
				if cc, err := cfg.CartCookie.Get(c); err == nil && cc != nil {
					if page, err := cfg.CartSvc.BuildCartPageFromCookie(c, cc); err == nil {
						preview = page
						previewSet = true
						qty = page.Count
					}
				}
			}
		}

		if !previewSet {
			// 1) Logged-in: use cached cart lookup (very efficient after first request)
			if u, ok := CurrentUser(c); ok && cfg.DB != nil {
				if n, err := cartQtyFromDBCached(c.Request.Context(), cfg.DB, c, u.ID); err == nil {
					qty = n
				}
			} else if cfg.CookieQty != nil {
				// 2) No DB cart (guest or DB empty): try cookie fallback
				if n, err := cfg.CookieQty(c); err == nil && n > 0 {
					qty = n
				}
			}
		}

		c.Set(cartPreviewKey, preview)
		c.Set(cartBadgeKey, qty)
		c.Next()
	}
}

func GetCartBadgeQty(c *gin.Context) int {
	v, ok := c.Get(cartBadgeKey)
	if !ok {
		return 0
	}
	n, _ := v.(int)
	return n
}

func GetCartPreview(c *gin.Context) view.CartPage {
	v, ok := c.Get(cartPreviewKey)
	if !ok {
		return view.CartPage{}
	}
	page, ok := v.(view.CartPage)
	if !ok {
		return view.CartPage{}
	}
	return page
}

// DB schema varsayımı (minimum):
// carts: id, user_id, updated_at
// cart_items: cart_id, quantity
//
// Tek SQL sorgusu: subquery ile en güncel cart'ı bul, qty'sini topla.
// Open cart yoksa COALESCE ile 0 döner.
// Performans: 2 query yerine 1 query → SSR header'da önemli.
func cartQtyFromDB(ctx context.Context, db *gorm.DB, userID string) (int, error) {
	const q = `SELECT COALESCE(SUM(ci.quantity), 0) AS qty
FROM cart_items ci
WHERE ci.cart_id = (
  SELECT c.id
  FROM carts c
  WHERE c.user_id = ? AND c.status = 'open'
  ORDER BY c.updated_at DESC
  LIMIT 1
);`

	var sum int64
	if err := db.WithContext(ctx).Raw(q, userID).Scan(&sum).Error; err != nil {
		return 0, err
	}
	if sum < 0 {
		sum = 0
	}
	return int(sum), nil
}

// Session cache helpers
// Note: Using gin.Context local storage for request-scoped cache.
// This avoids persisting to DB and keeps session lightweight.
const sessionActiveCartIDKey = "session_active_cart_id"

func sessionGetString(c *gin.Context, key string) (string, bool) {
	v, exists := c.Get(key)
	if !exists {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func sessionSetString(c *gin.Context, key, val string) {
	c.Set(key, val)
}

// Optimized cart qty fetch with session caching
func cartQtyFromDBCached(ctx context.Context, db *gorm.DB, c *gin.Context, userID string) (int, error) {
	// 1) Check session cache for active_cart_id
	if cartID, ok := sessionGetString(c, sessionActiveCartIDKey); ok && cartID != "" {
		// Cart ID is cached; just sum items (very cheap query)
		qty, err := sumCartQty(ctx, db, cartID)
		if err != nil {
			return 0, err
		}
		// Cart may be empty (qty=0) but cache stays valid
		return qty, nil
	}

	// 2) Cache miss: find user's active cart and cache it
	cartID, err := findActiveCartID(ctx, db, userID)
	if err != nil {
		return 0, err
	}
	if cartID == "" {
		return 0, nil
	}
	sessionSetString(c, sessionActiveCartIDKey, cartID)
	return sumCartQty(ctx, db, cartID)
}

// Find the most recent open cart for a user
func findActiveCartID(ctx context.Context, db *gorm.DB, userID string) (string, error) {
	const q = `
SELECT c.id
FROM carts c
WHERE c.user_id = ? AND c.status = 'open'
ORDER BY c.updated_at DESC
LIMIT 1;`
	var cartID string
	if err := db.WithContext(ctx).Raw(q, userID).Scan(&cartID).Error; err != nil {
		return "", err
	}
	return cartID, nil
}

// Sum quantities for a specific cart
func sumCartQty(ctx context.Context, db *gorm.DB, cartID string) (int, error) {
	const q = `SELECT COALESCE(SUM(quantity), 0) AS qty FROM cart_items WHERE cart_id = ?;`
	var sum int64
	if err := db.WithContext(ctx).Raw(q, cartID).Scan(&sum).Error; err != nil {
		return 0, err
	}
	if sum < 0 {
		sum = 0
	}
	return int(sum), nil
}

// Clear session cart cache (call after checkout/cart operations)
func ClearSessionCartCache(c *gin.Context) {
	sessionSetString(c, sessionActiveCartIDKey, "")
}
