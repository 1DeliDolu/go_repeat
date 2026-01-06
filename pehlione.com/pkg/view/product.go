package view

type ProductVariant struct {
	ID   string
	Name string
}

type ProductDetail struct {
	ID          string
	Name        string
	Slug        string
	Description string
	Price       string
	ImageURL    string
}

type ProductDetailPage struct {
	Product    ProductDetail
	Variants   []ProductVariant
	Highlights []string
}
