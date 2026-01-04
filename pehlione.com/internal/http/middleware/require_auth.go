package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/pkg/view"
)

// RequireAuth: giriş yoksa
// - SSR: flash + /login?return_to=... redirect
// - JSON: 401 JSON
func RequireAuth(flashCodec *flash.Codec) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := CurrentUser(c); ok {
			c.Next()
			return
		}

		// JSON isteyen client
		if WantsJSON(c) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":      "authentication required",
				"request_id": GetRequestID(c),
			})
			return
		}

		// SSR: login’e yönlendir, geri dönüş için request uri’yi taşı
		returnTo := c.Request.URL.RequestURI()
		SetFlashCookie(c, flashCodec, view.Flash{
			Kind:    view.FlashWarning,
			Message: "Devam etmek için giriş yapın.",
		})

		c.Redirect(http.StatusFound, "/login?return_to="+url.QueryEscape(returnTo))
		c.Abort()
	}
}
