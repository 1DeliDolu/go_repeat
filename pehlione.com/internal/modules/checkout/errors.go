package checkout

import "fmt"

type OutOfStockItem struct {
	VariantID string
	Requested int
	Available int
}

type OutOfStockError struct {
	Items []OutOfStockItem
}

func (e *OutOfStockError) Error() string {
	if len(e.Items) == 0 {
		return "out of stock"
	}
	it := e.Items[0]
	return fmt.Sprintf("out of stock: variant=%s requested=%d available=%d", it.VariantID, it.Requested, it.Available)
}
