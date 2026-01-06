package view

import "context"

type headerCtxKey struct{}

// HeaderCtx holds header view state: auth status, admin flag, CSRF token, cart qty.
type HeaderCtx struct {
	IsAuthed  bool
	IsAdmin   bool
	UserEmail string // optional: display in header
	CSRFToken string // required: for logout form
	CartQty   int    // number of items in cart
	Cart      CartPage
}

// WithHeaderCtx injects HeaderCtx into request context.
func WithHeaderCtx(ctx context.Context, h HeaderCtx) context.Context {
	return context.WithValue(ctx, headerCtxKey{}, h)
}

// HeaderCtxFrom extracts HeaderCtx from request context.
// Returns zero value if not found.
func HeaderCtxFrom(ctx context.Context) HeaderCtx {
	h, ok := ctx.Value(headerCtxKey{}).(HeaderCtx)
	if !ok {
		return HeaderCtx{}
	}
	return h
}
