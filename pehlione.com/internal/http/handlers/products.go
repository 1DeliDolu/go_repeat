package handlers

import (
	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/templates/pages"
)

// ProductsHandler handles product listing
type ProductsHandler struct{}

func NewProductsHandler() *ProductsHandler {
	return &ProductsHandler{}
}

// List returns the products listing page
func (h *ProductsHandler) List(c *gin.Context) {
	flash := middleware.GetFlash(c)
	headerCtx := middleware.BuildHeaderCtx(c)

	render.Component(c, 200, pages.Products(flash, headerCtx))
}
