package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/auth"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type AccountEditHandler struct {
	authRepo *auth.Repo
	Flash    *flash.Codec
}

func NewAccountEditHandler(authRepo *auth.Repo, flashCodec *flash.Codec) *AccountEditHandler {
	return &AccountEditHandler{authRepo: authRepo, Flash: flashCodec}
}

func (h *AccountEditHandler) Get(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	render.Component(c, http.StatusOK, pages.AccountEdit(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		user.Email,
		user.FirstName,
		user.LastName,
		user.PhoneE164,
		user.Address,
	))
}

func (h *AccountEditHandler) Post(c *gin.Context) {
	userCtx, ok := middleware.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Fetch the full user from the database
	user, err := h.authRepo.GetByID(c.Request.Context(), userCtx.ID)
	if err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account/edit", view.FlashError, "Profil yüklenemedi")
		return
	}

	// Parse form data
	if err := c.Request.ParseForm(); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account/edit", view.FlashError, "Form işlenemedi")
		return
	}

	email := c.PostForm("email")
	firstName := c.PostForm("first_name")
	lastName := c.PostForm("last_name")
	phone := c.PostForm("phone")
	address := c.PostForm("address")

	// Validate email
	if email == "" {
		render.RedirectWithFlash(c, h.Flash, "/account/edit", view.FlashError, "E-posta gereklidir")
		return
	}

	// Update user
	user.Email = email

	if firstName != "" {
		user.FirstName = &firstName
	} else {
		user.FirstName = nil
	}

	if lastName != "" {
		user.LastName = &lastName
	} else {
		user.LastName = nil
	}

	if phone != "" {
		user.PhoneE164 = &phone
	} else {
		user.PhoneE164 = nil
	}

	if address != "" {
		user.Address = &address
	} else {
		user.Address = nil
	}

	if err := h.authRepo.UpdateUser(c.Request.Context(), user); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/account/edit", view.FlashError, "Profil güncellenemedi")
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/account", view.FlashSuccess, "Profiliniz başarıyla güncellendi")
}
