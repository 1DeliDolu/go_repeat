package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"strings"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/auth"
	"pehlione.com/app/internal/modules/orders"
	"pehlione.com/app/internal/modules/products"
	"pehlione.com/app/internal/modules/users"
	"pehlione.com/app/internal/modules/wishlist"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type AccountHandler struct {
	authRepo          *auth.Repo
	Flash             *flash.Codec
	wishlist          *wishlist.Service
	products          products.Repository
	ordersRepo        *orders.Repo
	passwordChangeSvc *users.PasswordChangeService
}

func NewAccountHandler(authRepo *auth.Repo, flashCodec *flash.Codec, wsvc *wishlist.Service, prodRepo products.Repository, ordersRepo *orders.Repo) *AccountHandler {
	return &AccountHandler{authRepo: authRepo, Flash: flashCodec, wishlist: wsvc, products: prodRepo, ordersRepo: ordersRepo}
}

func (h *AccountHandler) SetPasswordChangeService(svc *users.PasswordChangeService) {
	h.passwordChangeSvc = svc
}

func (h *AccountHandler) Get(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	activeTab := c.DefaultQuery("tab", "account")

	// Get pagination for wishlist
	page := 1
	if p := c.Query("wishlist_page"); p != "" {
		if pv, err := strconv.Atoi(p); err == nil && pv > 0 {
			page = pv
		}
	}

	// Fetch all wishlist items
	allWishlistItems := []pages.WishlistItem{}
	displayCurrency := middleware.GetDisplayCurrency(c)
	if h.wishlist != nil && h.products != nil {
		items, err := h.wishlist.Items(c.Request.Context(), user.ID)
		if err == nil && len(items) > 0 {
			productIDs := make([]string, 0, len(items))
			for _, it := range items {
				productIDs = append(productIDs, it.ProductID)
			}

			prods, err := h.products.ListByIDs(c.Request.Context(), productIDs)
			if err == nil {
				for _, p := range prods {
					card := pages.WishlistItem{
						ProductID: p.ID,
						Title:     p.Name,
						Slug:      p.Slug,
						ImageURL:  "",
						Currency:  displayCurrency,
					}
					if len(p.Images) > 0 {
						card.ImageURL = p.Images[0].URL
					}
					if len(p.Variants) > 0 {
						card.PriceCents = int64(p.Variants[0].PriceCents)
					}
					allWishlistItems = append(allWishlistItems, card)
				}
			}
		}
	}

	// Paginate wishlist items (3 per page)
	pageSize := 3
	totalPages := 1
	if len(allWishlistItems) > 0 {
		totalPages = (len(allWishlistItems) + pageSize - 1) / pageSize
	}
	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * pageSize
	endOffset := offset + pageSize
	if endOffset > len(allWishlistItems) {
		endOffset = len(allWishlistItems)
	}

	var paginatedWishlist []pages.WishlistItem
	if offset < len(allWishlistItems) {
		paginatedWishlist = allWishlistItems[offset:endOffset]
	}

	// Fetch orders if needed
	ordersList := []view.AccountOrderListItem{}
	if activeTab == "orders" && h.ordersRepo != nil {
		result, err := h.ordersRepo.ListByUser(c.Request.Context(), orders.ListByUserParams{
			UserID:   user.ID,
			Page:     1,
			PageSize: 100,
			Status:   "",
		})
		if err == nil {
			for _, item := range result.Items {
				orderNum := item.Order.ID
				if len(orderNum) > 8 {
					orderNum = orderNum[:8]
				}

				ordersList = append(ordersList, view.AccountOrderListItem{
					ID:         item.Order.ID,
					Number:     orderNum,
					CreatedAt:  item.Order.CreatedAt,
					Status:     item.Order.Status,
					TotalCents: int64(item.Order.TotalCents),
					Currency:   item.Order.Currency,
					ItemCount:  item.Count,
					PaidAt:     item.Order.PaidAt,
				})
			}
		}
	}

	render.Component(c, http.StatusOK, pages.Account(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		user.Email,
		user.EmailVerifiedAt,
		user.FirstName,
		user.LastName,
		user.Address,
		paginatedWishlist,
		page,
		totalPages,
		ordersList,
		activeTab,
	))
}

func (h *AccountHandler) ChangePassword(c *gin.Context) {
	userCtx, ok := middleware.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	fullUser, err := h.authRepo.GetByID(c.Request.Context(), userCtx.ID)
	if err != nil {
		c.Error(apperr.Wrap(err))
		return
	}

	errorsMap := map[string]string{}
	current := strings.TrimSpace(c.PostForm("current_password"))
	newPass := strings.TrimSpace(c.PostForm("new_password"))
	confirm := strings.TrimSpace(c.PostForm("confirm_password"))

	if current == "" {
		errorsMap["current_password"] = "Mevcut şifreyi girin."
	}
	if len(newPass) < 6 {
		errorsMap["new_password"] = "Yeni şifre en az 6 karakter olmalı."
	}
	if newPass != confirm {
		errorsMap["confirm_password"] = "Şifreler eşleşmiyor."
	}

	if len(errorsMap) == 0 {
		if err := bcrypt.CompareHashAndPassword([]byte(fullUser.PasswordHash), []byte(current)); err != nil {
			errorsMap["current_password"] = "Mevcut şifre hatalı."
		}
	}

	if len(errorsMap) > 0 {
		// How to render errors? The form is on a different page now.
		// For now, let's just redirect back with a generic error flash.
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Şifre güncellenemedi, lütfen hataları kontrol edin.")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		c.Error(apperr.Wrap(err))
		return
	}

	// Use password change service to send confirmation email if available
	if h.passwordChangeSvc != nil {
		if err := h.passwordChangeSvc.StartPasswordChange(c.Request.Context(), fullUser.ID, fullUser.Email, string(hashed)); err != nil {
			render.RedirectWithFlash(c, h.Flash, "/account", view.FlashError, "Şifre onay e-postası gönderilemedi.")
			return
		}
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashInfo, "Şifre değişikliği e-postası gönderildi. Lütfen e-posta hesabınızı kontrol edin ve onaylayın.")
	} else {
		// Fallback: directly update password if service not available
		if err := h.authRepo.UpdatePassword(c.Request.Context(), fullUser.ID, string(hashed)); err != nil {
			c.Error(apperr.Wrap(err))
			return
		}
		render.RedirectWithFlash(c, h.Flash, "/account", view.FlashSuccess, "Şifreniz güncellendi.")
	}
}
