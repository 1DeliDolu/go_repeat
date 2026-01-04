package handlers

import (
	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/templates/pages"
)

// CartHandler handles the shopping cart page
type CartHandler struct{}

func NewCartHandler() *CartHandler {
	return &CartHandler{}
}

// Get returns the cart page
func (h *CartHandler) Get(c *gin.Context) {
	flash := middleware.GetFlash(c)
	render.Component(c, 200, pages.Cart(flash))
}
