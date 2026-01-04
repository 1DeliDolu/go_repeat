package products

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) List(ctx context.Context) ([]Product, error) {
	var items []Product
	err := r.db.WithContext(ctx).
		Order("updated_at DESC").
		Find(&items).Error
	return items, err
}

func (r *Repo) Get(ctx context.Context, id string) (Product, error) {
	var p Product
	err := r.db.WithContext(ctx).
		Preload("Variants", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC") }).
		Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		First(&p, "id = ?", id).Error
	return p, err
}

func (r *Repo) CreateProduct(ctx context.Context, name, slug, desc, status string) (Product, error) {
	p := Product{
		ID:          uuid.NewString(),
		Name:        name,
		Slug:        slug,
		Description: desc,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&p).Error; err != nil {
		return Product{}, err
	}
	return p, nil
}

func (r *Repo) UpdateProduct(ctx context.Context, id, name, slug, desc, status string) error {
	return r.db.WithContext(ctx).Model(&Product{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"name":        name,
			"slug":        slug,
			"description": desc,
			"status":      status,
			"updated_at":  time.Now(),
		}).Error
}

func (r *Repo) DeleteProduct(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&Product{}, "id = ?", id).Error
}

func (r *Repo) AddVariant(ctx context.Context, productID, sku string, optionsJSON []byte, priceCents int, currency string, stock int) (Variant, error) {
	v := Variant{
		ID:         uuid.NewString(),
		ProductID:  productID,
		SKU:        sku,
		Options:    optionsJSON,
		PriceCents: priceCents,
		Currency:   currency,
		Stock:      stock,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&v).Error; err != nil {
		return Variant{}, err
	}
	return v, nil
}

func (r *Repo) DeleteVariant(ctx context.Context, productID, variantID string) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND product_id = ?", variantID, productID).
		Delete(&Variant{}).Error
}

func (r *Repo) AddImage(ctx context.Context, productID, url string, position int) (Image, error) {
	im := Image{
		ID:         uuid.NewString(),
		ProductID:  productID,
		StorageKey: url,
		URL:        url,
		Position:   position,
		CreatedAt:  time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&im).Error; err != nil {
		return Image{}, err
	}
	return im, nil
}

func (r *Repo) UpdateVariant(ctx context.Context, productID, variantID string, priceCents int, currency string, stock int, optionsJSON []byte) error {
	return r.db.WithContext(ctx).Model(&Variant{}).
		Where("id = ? AND product_id = ?", variantID, productID).
		Updates(map[string]any{
			"price_cents":  priceCents,
			"currency":     currency,
			"stock":        stock,
			"options_json": optionsJSON,
			"updated_at":   time.Now(),
		}).Error
}

func (r *Repo) UpdateVariantSKU(ctx context.Context, productID, variantID string, newSKU string) error {
	return r.db.WithContext(ctx).Model(&Variant{}).
		Where("id = ? AND product_id = ?", variantID, productID).
		Updates(map[string]any{
			"sku":        newSKU,
			"updated_at": time.Now(),
		}).Error
}

func (r *Repo) AddImageWithKey(ctx context.Context, productID, storageKey, url string, position int) (Image, error) {
	im := Image{
		ID:         uuid.NewString(),
		ProductID:  productID,
		StorageKey: storageKey,
		URL:        url,
		Position:   position,
		CreatedAt:  time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&im).Error; err != nil {
		return Image{}, err
	}
	return im, nil
}

func (r *Repo) GetImage(ctx context.Context, productID, imageID string) (Image, error) {
	var im Image
	err := r.db.WithContext(ctx).First(&im, "id = ? AND product_id = ?", imageID, productID).Error
	return im, err
}

func (r *Repo) DeleteImage(ctx context.Context, productID, imageID string) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND product_id = ?", imageID, productID).
		Delete(&Image{}).Error
}

func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	return false
}
