package render

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/pkg/view"
)

func Component(c *gin.Context, status int, component templ.Component) {
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")

	// Inject header context (auth state, admin flag, CSRF token)
	// into request context so header.templ can read it via ctx
	h := middleware.BuildHeaderCtx(c)
	ctx := view.WithHeaderCtx(c.Request.Context(), h)

	if err := component.Render(ctx, c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
	}
}
