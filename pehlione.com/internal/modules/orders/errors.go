package orders

import "errors"

var (
	ErrCartEmpty          = errors.New("cart is empty")
	ErrCurrencyMismatch   = errors.New("currency mismatch in cart")
	ErrProductUnavailable = errors.New("product unavailable")
)
