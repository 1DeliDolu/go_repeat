package render

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/pkg/view"
)

func RedirectWithFlash(c *gin.Context, codec *flash.Codec, location string, kind view.FlashKind, msg string) {
	middleware.SetFlashCookie(c, codec, view.Flash{Kind: kind, Message: msg})
	c.Redirect(http.StatusFound, location)
}
