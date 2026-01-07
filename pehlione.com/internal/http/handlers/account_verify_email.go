package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/users"
	"pehlione.com/app/pkg/view"
)

type AccountVerifyEmailHandler struct {
	verifySvc *users.VerifyService
	Flash     *flash.Codec
}

func NewAccountVerifyEmailHandler(verifySvc *users.VerifyService, flashCodec *flash.Codec) *AccountVerifyEmailHandler {
	return &AccountVerifyEmailHandler{verifySvc: verifySvc, Flash: flashCodec}
}

func (h *AccountVerifyEmailHandler) SendVerificationEmail(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Send verification email
	if h.verifySvc != nil {
		if err := h.verifySvc.StartEmailVerification(c.Request.Context(), user.ID, user.Email); err != nil {
			render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Doğrulama e-postası gönderilemedi")
			return
		}
	}

	render.RedirectWithFlash(c, h.Flash, "/account", view.FlashSuccess, "Doğrulama e-postası gönderildi. Lütfen e-posta hesabınızı kontrol edin.")
}
