package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/cartcookie"
	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/cart"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

// CartHandler handles cart operations (GET /cart, POST /cart/add)
type CartHandler struct {
	DB      *gorm.DB
	Flash   *flash.Codec
	CK      *cartcookie.Codec
	CartSvc *cart.Service
}

func NewCartHandler(db *gorm.DB, flashCodec *flash.Codec, ck *cartcookie.Codec, svc *cart.Service) *CartHandler {
	return &CartHandler{DB: db, Flash: flashCodec, CK: ck, CartSvc: svc}
}

// Add handles POST /cart/add - adds item to cart and redirects to /cart
func (h *CartHandler) Add(c *gin.Context) {
	variantID := strings.TrimSpace(c.PostForm("variant_id"))
	qtyStr := strings.TrimSpace(c.PostForm("qty"))

	qty := 1
	if qtyStr != "" {
		if n, err := strconv.Atoi(qtyStr); err == nil && n > 0 && n <= 99 {
			qty = n
		}
	}

	if variantID == "" {
		render.RedirectWithFlash(c, h.Flash, "/products", view.FlashError, "Variant seçilemedi.")
		return
	}

	// Check if user is logged in
	if u, ok := middleware.CurrentUser(c); ok && u.ID != "" {
		// Logged-in user: add to DB cart
		log.Printf("CartAdd: authenticated user %s adding variant_id=%s qty=%d", u.ID, variantID, qty)
		cartRepo := cart.NewRepo(h.DB)

		// Get or create user's cart
		userCart, err := cartRepo.GetOrCreateUserCart(c.Request.Context(), u.ID)
		if err != nil {
			log.Printf("CartAdd: error getting cart for user %s: %v", u.ID, err)
			render.RedirectWithFlash(c, h.Flash, "/products", view.FlashError, "Sepete ekleme başarısız.")
			return
		}
		log.Printf("CartAdd: user cart ID=%s", userCart.ID)

		// Add item to cart
		if err := cartRepo.AddItem(c.Request.Context(), userCart.ID, variantID, qty); err != nil {
			log.Printf("CartAdd: error adding item variant_id=%s to cart %s: %v", variantID, userCart.ID, err)
			render.RedirectWithFlash(c, h.Flash, "/products", view.FlashError, "Sepete ekleme başarısız.")
			return
		}
		log.Printf("CartAdd: successfully added variant_id=%s qty=%d to user %s cart", variantID, qty, u.ID)

		// Clear cache
		middleware.ClearSessionCartCache(c)

		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashSuccess, "✓ Sepete eklendi.")
		return
	}

	// Guest user: add to cookie cart
	cc, err := h.CK.Get(c)
	if err != nil {
		log.Printf("CartAdd guest: error getting cookie: %v", err)
	}
	if cc == nil {
		cc = cartcookie.NewCart()
	}
	log.Printf("CartAdd guest: before AddItem, cart has %d items", len(cc.Items))
	cc.AddItem(variantID, qty)
	log.Printf("CartAdd guest: after AddItem, cart has %d items", len(cc.Items))
	h.CK.Set(c, cc)
	log.Printf("CartAdd guest: cookie set for variant_id=%s qty=%d", variantID, qty)

	render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashSuccess, "✓ Sepete eklendi.")
}

// Update handles POST /cart/items/update - updates item quantity
func (h *CartHandler) Update(c *gin.Context) {
	variantID := strings.TrimSpace(c.PostForm("variant_id"))
	qtyStr := strings.TrimSpace(c.PostForm("qty"))
	qty := 1
	if qtyStr != "" {
		if n, err := strconv.Atoi(qtyStr); err == nil {
			qty = n
		}
	}

	if variantID == "" {
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Ürün bulunamadı.")
		return
	}

	qty = clamp(qty, 0, 99)

	if u, ok := middleware.CurrentUser(c); ok && u.ID != "" {
		repo := cart.NewRepo(h.DB)
		userCart, err := repo.GetOrCreateUserCart(c.Request.Context(), u.ID)
		if err != nil {
			log.Printf("CartUpdate: error getting cart: %v", err)
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet güncellenemedi.")
			return
		}

		if err := repo.UpdateItemQty(c.Request.Context(), userCart.ID, variantID, qty); err != nil {
			log.Printf("CartUpdate: update item error: %v", err)
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Miktar güncellenemedi.")
			return
		}

		middleware.ClearSessionCartCache(c)
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashSuccess, "Miktar güncellendi.")
		return
	}

	cc, _ := h.CK.Get(c)
	if cc == nil {
		cc = cartcookie.NewCart()
	}
	cc.UpdateQuantity(variantID, qty)
	h.CK.Set(c, cc)
	render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashSuccess, "Miktar güncellendi.")
}

// Remove handles POST /cart/items/remove - removes item from cart
func (h *CartHandler) Remove(c *gin.Context) {
	variantID := strings.TrimSpace(c.PostForm("variant_id"))
	if variantID == "" {
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashWarning, "Ürün bulunamadı.")
		return
	}

	if u, ok := middleware.CurrentUser(c); ok && u.ID != "" {
		repo := cart.NewRepo(h.DB)
		userCart, err := repo.GetOrCreateUserCart(c.Request.Context(), u.ID)
		if err != nil {
			log.Printf("CartRemove: error getting cart: %v", err)
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Sepet güncellenemedi.")
			return
		}

		if err := repo.RemoveItem(c.Request.Context(), userCart.ID, variantID); err != nil {
			log.Printf("CartRemove: remove item error: %v", err)
			render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashError, "Ürün silinemedi.")
			return
		}

		middleware.ClearSessionCartCache(c)
		render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashSuccess, "Ürün sepetten çıkarıldı.")
		return
	}

	cc, _ := h.CK.Get(c)
	if cc == nil {
		cc = cartcookie.NewCart()
	}
	cc.RemoveItem(variantID)
	h.CK.Set(c, cc)
	render.RedirectWithFlash(c, h.Flash, "/cart", view.FlashSuccess, "Ürün sepetten çıkarıldı.")
}

// Get handles GET /cart - displays cart page
func (h *CartHandler) Get(c *gin.Context) {
	flash := middleware.GetFlash(c)
	displayCurrency := middleware.GetDisplayCurrency(c)
	svc := h.CartSvc
	if svc == nil {
		svc = cart.NewService(h.DB, nil)
	}

	// Check if user is logged in
	if u, ok := middleware.CurrentUser(c); ok && u.ID != "" {
		// Logged-in user: fetch from DB
		log.Printf("CartGet: authenticated user %s, fetching DB cart", u.ID)
		cartPage, err := svc.BuildCartPageForUser(c.Request.Context(), u.ID, displayCurrency)
		if err != nil {
			log.Printf("CartGet: error building page for user %s: %v", u.ID, err)
			render.Component(c, http.StatusOK, pages.Cart(flash, view.CartPage{Items: []view.CartItem{}}))
			return
		}
		log.Printf("CartGet: user %s has %d items in cart", u.ID, len(cartPage.Items))
		cartPage.CSRFToken = csrfTokenFrom(c)
		render.Component(c, http.StatusOK, pages.Cart(flash, cartPage))
		return
	}
	log.Printf("CartGet: user not authenticated, using guest cookie")

	// Guest user: fetch from cookie
	cc, err := h.CK.Get(c)
	if err != nil {
		log.Printf("CartGet: error getting cookie: %v", err)
	}
	if cc == nil {
		cc = cartcookie.NewCart()
	}
	log.Printf("CartGet: guest cart has %d items", len(cc.Items))
	for _, item := range cc.Items {
		log.Printf("  - variant_id=%s qty=%d", item.VariantID, item.Qty)
	}
	cartPage, err := svc.BuildCartPageFromCookie(c.Request.Context(), cc, displayCurrency)
	if err != nil {
		log.Printf("CartGet: error building guest cart: %v", err)
		render.Component(c, http.StatusOK, pages.Cart(flash, view.CartPage{Items: []view.CartItem{}}))
		return
	}
	cartPage.CSRFToken = csrfTokenFrom(c)

	render.Component(c, http.StatusOK, pages.Cart(flash, cartPage))
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
