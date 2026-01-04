package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/shared/apperr"
)

func Recovery(l *slog.Logger) gin.HandlerFunc {
	// stdout'a stack yazmak yerine structured log'a koyuyoruz.
	// isterseniz stack'i ayrı field olarak saklayın.
	_ = io.Discard
	_ = os.Stdout

	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		stack := debug.Stack()
		l.LogAttrs(c.Request.Context(), slog.LevelError, "panic_recovered",
			slog.String("request_id", GetRequestID(c)),
			slog.Any("panic", recovered),
			slog.String("stack", string(stack)),
		)

		Fail(c, apperr.Wrap(fmt.Errorf("panic: %v", recovered)))
	})
}
