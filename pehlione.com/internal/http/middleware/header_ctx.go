package middleware

import (
	"github.com/gin-gonic/gin"

	"pehlione.com/app/pkg/view"
)

// BuildHeaderCtx builds HeaderCtx from current request state.
// This is a single adaptation point for auth/admin/csrf logic.
func BuildHeaderCtx(c *gin.Context) view.HeaderCtx {
	h := view.HeaderCtx{
		CSRFToken: GetCSRFToken(c),
		CartQty:   GetCartBadgeQty(c),
		Cart:      GetCartPreview(c),
	}

	u, ok := CurrentUser(c)
	if !ok {
		return h // not authenticated
	}

	h.IsAuthed = true
	h.UserEmail = u.Email

	// Admin check: role == "admin"
	h.IsAdmin = (u.Role == "admin")

	return h
}
