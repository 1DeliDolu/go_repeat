package middleware

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/pkg/view"
)

// RequireAdmin:
// - login yoksa: login’e redirect (return_to ile) + flash
// - login var ama admin değilse: SSR -> home redirect + flash, JSON -> 403
func RequireAdmin(flashCodec *flash.Codec) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := CurrentUser(c)
		if !ok {
			// JSON isteyen client
			if WantsJSON(c) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":      "authentication required",
					"request_id": GetRequestID(c),
				})
				return
			}

			// SSR: login redirect
			returnTo := c.Request.URL.RequestURI()
			SetFlashCookie(c, flashCodec, view.Flash{
				Kind:    view.FlashWarning,
				Message: "Admin paneli için giriş yapın.",
			})
			c.Redirect(http.StatusFound, "/login?return_to="+url.QueryEscape(returnTo))
			c.Abort()
			return
		}

		if u.Role != "admin" {
			if WantsJSON(c) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":      "forbidden",
					"request_id": GetRequestID(c),
				})
				return
			}

			// SSR: home'a yönlendir + flash
			SetFlashCookie(c, flashCodec, view.Flash{
				Kind:    view.FlashError,
				Message: "Bu sayfaya erişim yetkiniz yok (admin gerekli).",
			})
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		c.Next()
	}
}
