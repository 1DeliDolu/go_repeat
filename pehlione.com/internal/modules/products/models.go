package products

import (
	"time"

	"gorm.io/datatypes"
)

type Product struct {
	ID          string    `gorm:"type:char(36);primaryKey"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Slug        string    `gorm:"type:varchar(255);not null;uniqueIndex:ux_products_slug"`
	Description string    `gorm:"type:text;not null"`
	Status      string    `gorm:"type:varchar(32);not null;default:active"`
	CreatedAt   time.Time `gorm:"type:datetime(3);not null"`
	UpdatedAt   time.Time `gorm:"type:datetime(3);not null"`

	Variants []Variant `gorm:"foreignKey:ProductID"`
	Images   []Image   `gorm:"foreignKey:ProductID"`
}

func (Product) TableName() string { return "products" }

type Variant struct {
	ID             string         `gorm:"type:char(36);primaryKey"`
	ProductID      string         `gorm:"type:char(36);not null;index:ix_variants_product_id"`
	SKU            string         `gorm:"type:varchar(64);not null;uniqueIndex:ux_variants_sku"`
	Options        datatypes.JSON `gorm:"type:json;not null"`
	PriceCents     int            `gorm:"not null"`
	CompareAtCents int            `gorm:"not null;default:0"`
	Currency       string         `gorm:"type:char(3);not null;default:EUR"`
	Stock          int            `gorm:"not null;default:0"`
	CreatedAt      time.Time      `gorm:"type:datetime(3);not null"`
	UpdatedAt      time.Time      `gorm:"type:datetime(3);not null"`
}

func (Variant) TableName() string { return "product_variants" }

type Image struct {
	ID         string    `gorm:"type:char(36);primaryKey"`
	ProductID  string    `gorm:"type:char(36);not null;index:ix_images_product_id"`
	StorageKey string    `gorm:"type:varchar(1024);not null"`
	URL        string    `gorm:"type:varchar(1024);not null"`
	Position   int       `gorm:"not null;default:0"`
	CreatedAt  time.Time `gorm:"type:datetime(3);not null"`
}

func (Image) TableName() string { return "product_images" }
