package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"log"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/templates/components"
)

// CartAddHandler handles adding items to the cart
type CartAddHandler struct {
	DB *gorm.DB
}

func NewCartAddHandler(db *gorm.DB) *CartAddHandler {
	return &CartAddHandler{DB: db}
}

// AddItem adds an item to the cart and returns updated badge
func (h *CartAddHandler) AddItem(c *gin.Context) {
	productID := c.PostForm("product_id")
	variantID := c.PostForm("variant_id")
	qtyStr := c.PostForm("quantity")

	if productID == "" {
		c.String(http.StatusBadRequest, "product_id required")
		return
	}

	// If variant_id not provided, use product_id as variant_id
	// (in a real system, you'd fetch the default variant)
	if variantID == "" {
		variantID = productID
	}

	qty := 1
	if qtyStr != "" {
		if q, err := strconv.Atoi(qtyStr); err == nil && q > 0 {
			qty = q
		}
	}

	// Get user ID from session
	userID, exists := c.Get("user_id")
	if !exists || userID == "" {
		// Guest user - would use session-based cart
		log.Printf("CartAdd: guest add product=%s variant=%s qty=%d", productID, variantID, qty)
		// For now, just return success with badge component
		middleware.ClearSessionCartCache(c)
		headerCtx := middleware.BuildHeaderCtx(c)
		render.Component(c, http.StatusOK, components.CartBadge(headerCtx))
		return
	}

	// Get or create cart for user
	cartRepo := cart.NewRepo(h.DB)
	userCart, err := cartRepo.GetOrCreateUserCart(c, userID.(string))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get cart")
		return
	}

	// Add item to cart
	if err := cartRepo.AddItem(c, userCart.ID, variantID, qty); err != nil {
		c.String(http.StatusInternalServerError, "Failed to add item")
		return
	}

	// Clear cache and fetch fresh badge count
	middleware.ClearSessionCartCache(c)
	headerCtx := middleware.BuildHeaderCtx(c)

	// Return updated badge component for HTMX to swap
	render.Component(c, http.StatusOK, components.CartBadge(headerCtx))
}
