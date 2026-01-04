package view

type AdminProductListItem struct {
	ID     string
	Name   string
	Slug   string
	Status string
}

type AdminVariant struct {
	ID         string
	SKU        string
	PriceCents int
	Currency   string
	Stock      int
	Options    string // JSON string
}

type AdminImage struct {
	ID       string
	URL      string
	Position int
}

type AdminProduct struct {
	ID          string
	Name        string
	Slug        string
	Description string
	Status      string
	Variants    []AdminVariant
	Images      []AdminImage
}
