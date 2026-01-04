package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/shared/apperr"
)

func WantsJSON(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		return true
	}
	return false
}

func Fail(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

func ErrorHandler(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() {
			return
		}
		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		status := apperr.HTTPStatus(err)
		publicMsg := apperr.PublicMessage(err)
		rid := GetRequestID(c)

		l.LogAttrs(c.Request.Context(), slog.LevelError, "request_failed",
			slog.String("request_id", rid),
			slog.Int("status", status),
			slog.Any("err", err),
		)

		if WantsJSON(c) {
			payload := gin.H{
				"error":      publicMsg,
				"request_id": rid,
			}
			if ae, ok := apperr.As(err); ok && len(ae.Fields) > 0 {
				payload["fields"] = ae.Fields
			}
			c.AbortWithStatusJSON(status, payload)
			return
		}

		// SSR: templ error page
		c.Abort()
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(status, fmt.Sprintf("<html><body><h1>%d %s</h1><p>%s</p><p>Request ID: %s</p></body></html>",
			status, http.StatusText(status), publicMsg, rid))
	}
}
