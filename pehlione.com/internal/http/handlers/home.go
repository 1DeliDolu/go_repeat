package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/templates/pages"
)

func Component(c *gin.Context, status int, component templ.Component) {
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
	}
}

func Home(c *gin.Context) {
	flash := middleware.GetFlash(c)
	Component(c, http.StatusOK, pages.Home(flash))
}
