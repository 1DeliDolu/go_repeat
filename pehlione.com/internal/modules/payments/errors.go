package payments

import "errors"

var (
	ErrOrderNotPayable = errors.New("order not payable")
	ErrForbidden       = errors.New("forbidden")
)
