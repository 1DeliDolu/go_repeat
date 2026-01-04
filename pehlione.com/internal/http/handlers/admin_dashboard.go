package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/templates/pages"
)

func AdminDashboard(c *gin.Context) {
	u, _ := middleware.CurrentUser(c) // RequireAdmin garanti eder

	render.Component(c, http.StatusOK, pages.AdminDashboard(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		u.Email,
	))
}
