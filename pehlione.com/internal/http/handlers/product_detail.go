package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/products"
	"pehlione.com/app/pkg/view"
	pages "pehlione.com/app/templates/pages"
)

// ProductDetailHandler handles product detail page
type ProductDetailHandler struct {
	db *gorm.DB
}

func NewProductDetailHandler(db *gorm.DB) *ProductDetailHandler {
	return &ProductDetailHandler{db: db}
}

// Detail returns the product detail page
func (h *ProductDetailHandler) Detail(c *gin.Context) {
	flash := middleware.GetFlash(c)

	slug := c.Param("slug")

	// Fetch product by slug
	var product products.Product
	if err := h.db.Where("slug = ?", slug).First(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Ürün bulunamadı"})
			return
		}
		c.JSON(500, gin.H{"error": "Veritabanı hatası"})
		return
	}

	// Fetch variants
	var dbVariants []products.Variant
	h.db.Where("product_id = ?", product.ID).Find(&dbVariants)

	// Convert to view variants
	variants := make([]view.ProductVariant, len(dbVariants))
	for i, v := range dbVariants {
		variants[i] = view.ProductVariant{
			ID:   v.ID,
			Name: v.SKU,
		}
	}

	// Get main image
	var mainImage products.Image
	h.db.Where("product_id = ?", product.ID).Order("position ASC").First(&mainImage)

	// Build product detail page
	p := view.ProductDetailPage{
		Product: view.ProductDetail{
			ID:          product.ID,
			Name:        product.Name,
			Slug:        product.Slug,
			Description: product.Description,
			Price:       "TBD", // Price will be calculated from variants
			ImageURL:    mainImage.URL,
		},
		Variants:   variants,
		Highlights: []string{}, // İleride eklenebilir
	}

	render.Component(c, 200, pages.Product(flash, p))
}
