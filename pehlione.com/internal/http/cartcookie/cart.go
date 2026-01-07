package cartcookie

import (
	"encoding/json"
)

type Item struct {
	VariantID string `json:"variant_id"`
	Qty       int    `json:"qty"`
}

type Cart struct {
	Items []Item `json:"items"`
}

func NewCart() *Cart {
	return &Cart{Items: []Item{}}
}

// AddItem ekler veya quantity'yi artırır
func (c *Cart) AddItem(variantID string, qty int) {
	if c == nil || qty < 1 {
		return
	}
	for i := range c.Items {
		if c.Items[i].VariantID == variantID {
			c.Items[i].Qty += qty
			return
		}
	}
	c.Items = append(c.Items, Item{VariantID: variantID, Qty: qty})
}

// UpdateQuantity sets quantity or removes if qty <=0
func (c *Cart) UpdateQuantity(variantID string, qty int) {
	if c == nil {
		return
	}
	for i := range c.Items {
		if c.Items[i].VariantID == variantID {
			if qty <= 0 {
				c.Items = append(c.Items[:i], c.Items[i+1:]...)
				return
			}
			c.Items[i].Qty = qty
			return
		}
	}
	if qty > 0 {
		c.Items = append(c.Items, Item{VariantID: variantID, Qty: qty})
	}
}

// RemoveItem deletes an item by variant id
func (c *Cart) RemoveItem(variantID string) {
	if c == nil {
		return
	}
	for i := range c.Items {
		if c.Items[i].VariantID == variantID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return
		}
	}
}

// ToJSON serializes cart
func (c *Cart) ToJSON() string {
	if c == nil {
		c = NewCart()
	}
	if c.Items == nil {
		c.Items = []Item{}
	}
	b, _ := json.Marshal(c)
	return string(b)
}

// FromJSON deserializes cart
func FromJSON(s string) *Cart {
	var c Cart
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return NewCart()
	}
	if c.Items == nil {
		c.Items = []Item{}
	}
	return &c
}
