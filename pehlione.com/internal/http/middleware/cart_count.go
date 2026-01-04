package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const cartCountKey = "cart_count"

type CartCountCfg struct {
	CookieName string
}

// Cookie payload için "best effort" parse:
// - URL-escaped JSON
// - raw JSON
// - base64url(JSON)
type cartCookiePayload struct {
	Items []struct {
		Qty int `json:"qty"`
	} `json:"items"`
}

func CartCount(cfg CartCountCfg) gin.HandlerFunc {
	name := strings.TrimSpace(cfg.CookieName)
	if name == "" {
		name = "pehlione_cart"
	}

	return func(c *gin.Context) {
		n := 0

		if raw, err := c.Cookie(name); err == nil && raw != "" {
			if qty, ok := tryParseCartQty(raw); ok {
				n = qty
			}
		}

		c.Set(cartCountKey, n)
		c.Next()
	}
}

func GetCartCount(c *gin.Context) int {
	v, ok := c.Get(cartCountKey)
	if !ok {
		return 0
	}
	n, _ := v.(int)
	return n
}

func tryParseCartQty(raw string) (int, bool) {
	// 1) URL decode dene
	s := raw
	if u, err := url.QueryUnescape(raw); err == nil && u != "" {
		s = u
	}

	// 2) JSON dene
	if qty, ok := parseCartQtyJSON([]byte(s)); ok {
		return qty, true
	}

	// 3) base64url dene
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		if qty, ok := parseCartQtyJSON(b); ok {
			return qty, true
		}
	}

	return 0, false
}

func parseCartQtyJSON(b []byte) (int, bool) {
	var p cartCookiePayload
	if err := json.Unmarshal(b, &p); err != nil {
		return 0, false
	}
	sum := 0
	for _, it := range p.Items {
		if it.Qty > 0 {
			sum += it.Qty
		}
	}
	return sum, true
}
