package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/templates/components"
)

type CartBadgeHandler struct {
	DB *gorm.DB
}

func NewCartBadgeHandler(db *gorm.DB) *CartBadgeHandler {
	return &CartBadgeHandler{DB: db}
}

// GetBadge returns the cart badge HTML (HTMX endpoint)
// Clears session cache to force fresh count fetch
func (h *CartBadgeHandler) GetBadge(c *gin.Context) {
	// Clear session cache (force fresh cart lookup on next badge render)
	middleware.ClearSessionCartCache(c)

	// Rebuild HeaderCtx with fresh cart count
	headerCtx := middleware.BuildHeaderCtx(c)

	// Render badge component
	render.Component(c, http.StatusOK, components.CartBadge(headerCtx))
}
