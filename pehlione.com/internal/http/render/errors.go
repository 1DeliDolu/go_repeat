package render

import (
	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/templates/pages"
)

func ErrorPage(c *gin.Context, status int, msg string, requestID string) {
	flash := middleware.GetFlash(c)
	Component(c, status, pages.Error(status, msg, requestID, flash))
}
