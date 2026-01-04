package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/pkg/view"
)

const CtxKeyFlash = "flash"

// FlashMiddleware: cookie'den flash okur, context'e koyar ve cookie'yi siler (tek kullanımlık).
func FlashMiddleware(codec *flash.Codec) gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, err := c.Cookie(codec.CookieName); err == nil && v != "" {
			if f, err := codec.Decode(v); err == nil {
				c.Set(CtxKeyFlash, f)
			}
			// Her durumda temizle: geçersizse de tekrar denemesin
			clearCookie(c, codec.CookieName, codec.Secure)
		}
		c.Next()
	}
}

func GetFlash(c *gin.Context) *view.Flash {
	if v, ok := c.Get(CtxKeyFlash); ok {
		if f, ok := v.(*view.Flash); ok {
			return f
		}
	}
	return nil
}

func SetFlashCookie(c *gin.Context, codec *flash.Codec, f view.Flash) {
	val, err := codec.Encode(f)
	if err != nil {
		return
	}
	// Path "/" + HttpOnly + SameSite Lax SSR için doğru default
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(codec.CookieName, val, codec.CookieMaxAge(), "/", "", codec.Secure, true)
}

func clearCookie(c *gin.Context, name string, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, "", -1, "/", "", secure, true)
}
