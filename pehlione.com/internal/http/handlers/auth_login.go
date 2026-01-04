package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/http/validation"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type LoginHandler struct {
	Flash *flash.Codec
}

func NewLoginHandler(f *flash.Codec) *LoginHandler {
	return &LoginHandler{Flash: f}
}

func (h *LoginHandler) Get(c *gin.Context) {
	returnTo := normalizeReturnTo(c.Query("return_to"))
	render.Component(c, http.StatusOK, pages.Login(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		returnTo,
		view.LoginForm{},
		nil,
		"",
	))
}

func (h *LoginHandler) Post(c *gin.Context) {
	returnTo := normalizeReturnTo(c.PostForm("return_to"))

	var in loginInput
	if err := c.ShouldBind(&in); err != nil {
		errs := validation.FromBindError(err, &in)
		// SSR: aynı sayfa 400 ile render
		render.Component(c, http.StatusBadRequest, pages.Login(
			middleware.GetFlash(c),
			middleware.GetCSRFToken(c),
			returnTo,
			view.LoginForm{Email: in.Email},
			errs,
			"",
		))
		return
	}

	// Burada gerçek auth daha sonra gelecek:
	// - userRepo.GetByEmail
	// - password verify
	// - session create
	//

	// Şimdilik demo: sadece domain benzeri kontrol:
	if !strings.HasSuffix(strings.ToLower(in.Email), "@example.com") {
		// "credentials" hatası: field değil page-level message
		render.Component(c, http.StatusUnauthorized, pages.Login(
			middleware.GetFlash(c),
			middleware.GetCSRFToken(c),
			returnTo,
			view.LoginForm{Email: in.Email},
			nil,
			"E-posta veya şifre hatalı.",
		))
		return
	}

	// ...existing code...

	// Başarılı: redirect + flash
	dest := "/"
	if returnTo != "" {
		dest = returnTo
	}
	render.RedirectWithFlash(c, h.Flash, dest, view.FlashSuccess, "Giriş başarılı.")
}
