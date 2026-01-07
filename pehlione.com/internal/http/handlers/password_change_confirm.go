package handlers

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/auth"
	"pehlione.com/app/internal/modules/users"
	"pehlione.com/app/pkg/view"
)

type PasswordChangeConfirmHandler struct {
	authRepo          *auth.Repo
	passwordChangeSvc *users.PasswordChangeService
	Flash             *flash.Codec
}

func NewPasswordChangeConfirmHandler(authRepo *auth.Repo, passwordChangeSvc *users.PasswordChangeService, flashCodec *flash.Codec) *PasswordChangeConfirmHandler {
	return &PasswordChangeConfirmHandler{authRepo: authRepo, passwordChangeSvc: passwordChangeSvc, Flash: flashCodec}
}

func (h *PasswordChangeConfirmHandler) ConfirmPasswordChange(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Geçersiz veya eksik token.")
		return
	}

	// Hash the token to look it up
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// Get the password change record and new password hash
	userID, newPasswordHash, err := h.passwordChangeSvc.ConfirmPasswordChange(c.Request.Context(), tokenHash)
	if err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Geçersiz veya süresi dolmuş onay bağlantısı.")
		return
	}

	// Apply the password change
	if err := h.authRepo.UpdatePassword(c.Request.Context(), userID, newPasswordHash); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Şifre güncellenemedi.")
		return
	}

	// Mark the password change as used
	if err := h.passwordChangeSvc.ApplyPasswordChange(c.Request.Context(), tokenHash, userID); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Şifre onaylanırken bir hata oluştu.")
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/account", view.FlashSuccess, "Şifreniz başarıyla değiştirildi.")
}

func (h *PasswordChangeConfirmHandler) CancelPasswordChange(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Geçersiz veya eksik token.")
		return
	}

	// Hash the token to look it up
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// Cancel the password change
	if err := h.passwordChangeSvc.CancelPasswordChange(c.Request.Context(), tokenHash); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Şifre değişikliği iptal edilemedi.")
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/account", view.FlashInfo, "Şifre değişikliği iptal edildi.")
}
