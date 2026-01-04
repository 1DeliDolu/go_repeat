package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/templates/pages"
)

func Account(c *gin.Context) {
	u, _ := middleware.CurrentUser(c) // RequireAuth zaten garanti ediyor
	render.Component(c, http.StatusOK, pages.Account(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		u.Email,
	))
}
