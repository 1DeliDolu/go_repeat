package products

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	ListActive(ctx context.Context, limit, offset int) ([]Product, error)
	GetBySlug(ctx context.Context, slug string) (Product, error)
}

type GormRepo struct {
	db *gorm.DB
}

func NewGormRepo(db *gorm.DB) *GormRepo {
	return &GormRepo{db: db}
}

func (r *GormRepo) ListActive(ctx context.Context, limit, offset int) ([]Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	var items []Product
	err := r.db.WithContext(ctx).
		Model(&Product{}).
		Where("status = ?", "active").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("position asc, id asc")
		}).
		Preload("Variants", func(db *gorm.DB) *gorm.DB {
			return db.Order("id asc")
		}).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	return items, err
}

func (r *GormRepo) GetBySlug(ctx context.Context, slug string) (Product, error) {
	var p Product
	err := r.db.WithContext(ctx).
		Model(&Product{}).
		Where("slug = ? AND status = ?", slug, "active").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("position asc, id asc")
		}).
		Preload("Variants", func(db *gorm.DB) *gorm.DB {
			return db.Order("id asc")
		}).
		First(&p).Error
	return p, err
}
