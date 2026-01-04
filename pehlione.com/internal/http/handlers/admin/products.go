package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/http/validation"
	"pehlione.com/app/internal/modules/products"
	"pehlione.com/app/internal/shared/apperr"
	"pehlione.com/app/internal/shared/slug"
	"pehlione.com/app/internal/storage"
	"pehlione.com/app/pkg/view"
	"pehlione.com/app/templates/pages"
)

type ProductsHandler struct {
	DB    *gorm.DB
	Flash *flash.Codec
	Store storage.Storage

	AllowSKUChange bool
}

func NewProductsHandler(db *gorm.DB, fl *flash.Codec, st storage.Storage) *ProductsHandler {
	return &ProductsHandler{DB: db, Flash: fl, Store: st, AllowSKUChange: os.Getenv("ALLOW_VARIANT_SKU_CHANGE") == "true"}
}

// ---------- List ----------
func (h *ProductsHandler) List(c *gin.Context) {
	repo := products.NewRepo(h.DB)
	items, err := repo.List(c.Request.Context())
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	out := make([]view.AdminProductListItem, 0, len(items))
	for _, p := range items {
		out = append(out, view.AdminProductListItem{
			ID:     p.ID,
			Name:   p.Name,
			Slug:   p.Slug,
			Status: p.Status,
		})
	}

	render.Component(c, http.StatusOK, pages.AdminProductsList(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		out,
	))
}

// ---------- Create ----------
func (h *ProductsHandler) New(c *gin.Context) {
	render.Component(c, http.StatusOK, pages.AdminProductForm(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		view.AdminProduct{Status: "active"},
		nil,
		"",
		false,
	))
}

type productInput struct {
	Name        string `form:"name" binding:"required,min=2,max=255"`
	Slug        string `form:"slug" binding:"omitempty,min=2,max=255"`
	Description string `form:"description" binding:"omitempty,max=5000"`
	Status      string `form:"status" binding:"required,oneof=active hidden"`
}

func (h *ProductsHandler) Create(c *gin.Context) {
	var in productInput
	if err := c.ShouldBind(&in); err != nil {
		errs := validation.FromBindError(err, &in)
		render.Component(c, http.StatusBadRequest, pages.AdminProductForm(
			middleware.GetFlash(c),
			middleware.GetCSRFToken(c),
			view.AdminProduct{Name: in.Name, Slug: in.Slug, Description: in.Description, Status: in.Status},
			errs,
			"",
			false,
		))
		return
	}

	if strings.TrimSpace(in.Slug) == "" {
		in.Slug = slug.FromName(in.Name)
	}

	repo := products.NewRepo(h.DB)
	p, err := repo.CreateProduct(c.Request.Context(), in.Name, in.Slug, in.Description, in.Status)
	if err != nil {
		if products.IsDuplicateKey(err) {
			render.Component(c, http.StatusConflict, pages.AdminProductForm(
				middleware.GetFlash(c),
				middleware.GetCSRFToken(c),
				view.AdminProduct{Name: in.Name, Slug: in.Slug, Description: in.Description, Status: in.Status},
				validation.FieldErrors{"slug": "Bu slug zaten kullanılıyor."},
				"",
				false,
			))
			return
		}
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+p.ID+"/edit", view.FlashSuccess, "Ürün oluşturuldu.")
}

// ---------- Edit ----------
func (h *ProductsHandler) Edit(c *gin.Context) {
	id := c.Param("id")
	repo := products.NewRepo(h.DB)

	p, err := repo.Get(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			middleware.Fail(c, apperr.NotFoundErr("Ürün bulunamadı."))
			return
		}
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	vm := toAdminProductVM(p)
	render.Component(c, http.StatusOK, pages.AdminProductForm(
		middleware.GetFlash(c),
		middleware.GetCSRFToken(c),
		vm,
		nil,
		"",
		true,
	))
}

func (h *ProductsHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var in productInput
	if err := c.ShouldBind(&in); err != nil {
		errs := validation.FromBindError(err, &in)
		render.Component(c, http.StatusBadRequest, pages.AdminProductForm(
			middleware.GetFlash(c),
			middleware.GetCSRFToken(c),
			view.AdminProduct{ID: id, Name: in.Name, Slug: in.Slug, Description: in.Description, Status: in.Status},
			errs,
			"",
			true,
		))
		return
	}

	if strings.TrimSpace(in.Slug) == "" {
		in.Slug = slug.FromName(in.Name)
	}

	repo := products.NewRepo(h.DB)
	if err := repo.UpdateProduct(c.Request.Context(), id, in.Name, in.Slug, in.Description, in.Status); err != nil {
		if products.IsDuplicateKey(err) {
			render.Component(c, http.StatusConflict, pages.AdminProductForm(
				middleware.GetFlash(c),
				middleware.GetCSRFToken(c),
				view.AdminProduct{ID: id, Name: in.Name, Slug: in.Slug, Description: in.Description, Status: in.Status},
				validation.FieldErrors{"slug": "Bu slug zaten kullanılıyor."},
				"",
				true,
			))
			return
		}
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashSuccess, "Ürün güncellendi.")
}

func (h *ProductsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	repo := products.NewRepo(h.DB)

	if err := repo.DeleteProduct(c.Request.Context(), id); err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}
	render.RedirectWithFlash(c, h.Flash, "/admin/products", view.FlashSuccess, "Ürün silindi.")
}

// ---------- Variants ----------
type variantInput struct {
	SKU        string `form:"sku" binding:"required,min=2,max=64"`
	PriceCents int    `form:"price_cents" binding:"required,min=0"`
	Currency   string `form:"currency" binding:"required,len=3"`
	Stock      int    `form:"stock" binding:"required,min=0"`
	Options    string `form:"options_json" binding:"omitempty"`
}

func (h *ProductsHandler) AddVariant(c *gin.Context) {
	id := c.Param("id")

	var in variantInput
	if err := c.ShouldBind(&in); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashError, "Variant formu geçersiz.")
		return
	}

	opts := strings.TrimSpace(in.Options)
	if opts == "" {
		opts = "{}"
	}
	if !json.Valid([]byte(opts)) {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashError, "options_json geçerli JSON olmalı.")
		return
	}

	repo := products.NewRepo(h.DB)
	_, err := repo.AddVariant(c.Request.Context(), id, in.SKU, []byte(opts), in.PriceCents, strings.ToUpper(in.Currency), in.Stock)
	if err != nil {
		if products.IsDuplicateKey(err) {
			render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashError, "SKU zaten kullanılıyor.")
			return
		}
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashSuccess, "Variant eklendi.")
}

func (h *ProductsHandler) DeleteVariant(c *gin.Context) {
	id := c.Param("id")
	vid := c.Param("vid")

	repo := products.NewRepo(h.DB)
	if err := repo.DeleteVariant(c.Request.Context(), id, vid); err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashSuccess, "Variant silindi.")
}

// ---------- Images ----------
type imageInput struct {
	URL      string `form:"url" binding:"required,min=5,max=1024"`
	Position int    `form:"position" binding:"required,min=0"`
}

func (h *ProductsHandler) AddImage(c *gin.Context) {
	id := c.Param("id")

	var in imageInput
	if err := c.ShouldBind(&in); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashError, "Görsel formu geçersiz.")
		return
	}

	// Legacy: allow adding by URL (treat URL as storage key/url)
	repo := products.NewRepo(h.DB)
	if _, err := repo.AddImage(c.Request.Context(), id, in.URL, in.Position); err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashSuccess, "Görsel eklendi.")
}

func (h *ProductsHandler) DeleteImage(c *gin.Context) {
	id := c.Param("id")
	iid := c.Param("iid")

	repo := products.NewRepo(h.DB)

	im, err := repo.GetImage(c.Request.Context(), id, iid)
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	// delete from storage first
	if err := h.Store.Delete(c.Request.Context(), im.StorageKey); err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	if err := repo.DeleteImage(c.Request.Context(), id, iid); err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+id+"/edit", view.FlashSuccess, "Görsel silindi.")
}

// UpdateVariant updates price/currency/stock/options
func (h *ProductsHandler) UpdateVariant(c *gin.Context) {
	pid := c.Param("id")
	vid := c.Param("vid")

	type inT struct {
		PriceCents int    `form:"price_cents" binding:"required,min=0"`
		Currency   string `form:"currency" binding:"required,len=3"`
		Stock      int    `form:"stock" binding:"required,min=0"`
		Options    string `form:"options_json" binding:"omitempty"`
	}
	var in inT
	if err := c.ShouldBind(&in); err != nil {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashError, "Variant update formu geçersiz.")
		return
	}

	opts := strings.TrimSpace(in.Options)
	if opts == "" {
		opts = "{}"
	}
	if !json.Valid([]byte(opts)) {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashError, "Variant options_json geçerli JSON olmalı.")
		return
	}

	repo := products.NewRepo(h.DB)
	if err := repo.UpdateVariant(
		c.Request.Context(),
		pid,
		vid,
		in.PriceCents,
		strings.ToUpper(in.Currency),
		in.Stock,
		[]byte(opts),
	); err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashSuccess, "Variant güncellendi.")
}

// UpdateVariantSKU handles SKU change when enabled
func (h *ProductsHandler) UpdateVariantSKU(c *gin.Context) {
	pid := c.Param("id")
	vid := c.Param("vid")

	if !h.AllowSKUChange {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashError, "SKU değişimi kapalı (ALLOW_VARIANT_SKU_CHANGE=false).")
		return
	}

	type inT struct {
		NewSKU  string `form:"new_sku" binding:"required,min=2,max=64"`
		Confirm string `form:"confirm_sku_change" binding:"required"`
	}
	var in inT
	if err := c.ShouldBind(&in); err != nil || in.Confirm != "1" {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashError, "SKU değişimi için onay gerekli.")
		return
	}

	repo := products.NewRepo(h.DB)
	if err := repo.UpdateVariantSKU(c.Request.Context(), pid, vid, in.NewSKU); err != nil {
		if products.IsDuplicateKey(err) {
			render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashError, "SKU zaten kullanılıyor.")
			return
		}
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashSuccess, "SKU güncellendi.")
}

// UploadImage handles multipart upload and stores via configured storage
func (h *ProductsHandler) UploadImage(c *gin.Context) {
	pid := c.Param("id")

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)

	file, err := c.FormFile("image")
	if err != nil {
		render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashError, "Dosya seçiniz (image).")
		return
	}

	posStr := strings.TrimSpace(c.PostForm("position"))
	position := 0
	if posStr != "" {
		if n, err := strconv.Atoi(posStr); err == nil && n >= 0 {
			position = n
		}
	}

	f, err := file.Open()
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}
	defer f.Close()

	ct := file.Header.Get("Content-Type")

	putRes, err := h.Store.Put(c.Request.Context(), f, storage.PutInput{
		Filename:    file.Filename,
		ContentType: ct,
		Size:        file.Size,
	})
	if err != nil {
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	repo := products.NewRepo(h.DB)
	if _, err := repo.AddImageWithKey(c.Request.Context(), pid, putRes.Key, putRes.URL, position); err != nil {
		_ = h.Store.Delete(c.Request.Context(), putRes.Key)
		middleware.Fail(c, apperr.Wrap(err))
		return
	}

	render.RedirectWithFlash(c, h.Flash, "/admin/products/"+pid+"/edit", view.FlashSuccess, "Görsel yüklendi.")
}

// ---------- helpers ----------
func toAdminProductVM(p products.Product) view.AdminProduct {
	vm := view.AdminProduct{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Status:      p.Status,
	}
	for _, v := range p.Variants {
		vm.Variants = append(vm.Variants, view.AdminVariant{
			ID:         v.ID,
			SKU:        v.SKU,
			PriceCents: v.PriceCents,
			Currency:   v.Currency,
			Stock:      v.Stock,
			Options:    string(v.Options),
		})
	}
	for _, im := range p.Images {
		vm.Images = append(vm.Images, view.AdminImage{
			ID:       im.ID,
			URL:      im.URL,
			Position: im.Position,
		})
	}
	return vm
}
