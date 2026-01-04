package orders

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pehlione.com/app/internal/modules/checkout"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CreateFromCartInput struct {
	CartID string

	UserID     *string
	GuestEmail *string

	// idempotency: sadece user için anlamlı (migration unique: user_id + key)
	IdempotencyKey *string

	// basit totals (şimdilik dışarıdan veya hesaplayarak)
	TaxCents      int
	ShippingCents int
	DiscountCents int

	ShippingAddressJSON []byte // optional
	BillingAddressJSON  []byte // optional
}

type CreateFromCartResult struct {
	OrderID       string
	Status        string
	Currency      string
	TotalCents    int
	SubtotalCents int
	Idempotent    bool // existing order döndüyse true
}

func (s *Service) CreateFromCart(ctx context.Context, in CreateFromCartInput) (CreateFromCartResult, error) {
	if in.CartID == "" {
		return CreateFromCartResult{}, ErrCartEmpty
	}

	var out CreateFromCartResult

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 1) Idempotency: user + key varsa önce mevcut order var mı bak
		if in.UserID != nil && in.IdempotencyKey != nil && *in.IdempotencyKey != "" {
			var existing Order
			err := tx.WithContext(ctx).First(&existing, "user_id = ? AND idempotency_key = ?", *in.UserID, *in.IdempotencyKey).Error
			if err == nil {
				out = CreateFromCartResult{
					OrderID:       existing.ID,
					Status:        existing.Status,
					Currency:      existing.Currency,
					TotalCents:    existing.TotalCents,
					SubtotalCents: existing.SubtotalCents,
					Idempotent:    true,
				}
				return nil
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		// 2) Cart items oku
		type CartItemRow struct {
			VariantID string `gorm:"column:variant_id"`
			Qty       int    `gorm:"column:quantity"`
		}
		var items []CartItemRow
		if err := tx.WithContext(ctx).
			Table("cart_items").
			Select("variant_id, quantity").
			Where("cart_id = ?", in.CartID).
			Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrCartEmpty
		}

		// qty map + lines
		want := map[string]int{}
		for _, it := range items {
			q := it.Qty
			if q < 1 {
				q = 1
			}
			want[it.VariantID] += q
		}

		ids := make([]string, 0, len(want))
		lines := make([]checkout.StockLine, 0, len(want))
		for id, q := range want {
			ids = append(ids, id)
			lines = append(lines, checkout.StockLine{VariantID: id, Qty: q})
		}
		sort.Strings(ids)

		// 3) Variant snapshot'u aynı tx içinde kilitle (FOR UPDATE) + fiyat/sku/options al
		type VariantSnap struct {
			ID         string `gorm:"column:id"`
			ProductID  string `gorm:"column:product_id"`
			SKU        string `gorm:"column:sku"`
			Options    []byte `gorm:"column:options_json"`
			PriceCents int    `gorm:"column:price_cents"`
			Currency   string `gorm:"column:currency"`
			Stock      int    `gorm:"column:stock"`
		}
		var vs []VariantSnap

		if err := tx.WithContext(ctx).
			Table("product_variants").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", ids).
			Order("id ASC").
			Find(&vs).Error; err != nil {
			return err
		}

		vmap := make(map[string]VariantSnap, len(vs))
		productIDs := make([]string, 0, len(vs))
		for _, v := range vs {
			vmap[v.ID] = v
			productIDs = append(productIDs, v.ProductID)
		}

		// basic existence check
		for _, id := range ids {
			if _, ok := vmap[id]; !ok {
				return ErrProductUnavailable
			}
		}

		// currency consistency
		currency := vmap[ids[0]].Currency
		for _, id := range ids[1:] {
			if vmap[id].Currency != currency {
				return ErrCurrencyMismatch
			}
		}

		// 4) STOCK DEDUCT (FOR UPDATE + validate + update) — istenen sıradaki kritik adım
		if err := checkout.DeductStockInTx(ctx, tx, lines); err != nil {
			return err // OutOfStockError buradan geçer
		}

		// 5) Product name snapshot
		type ProdRow struct {
			ID     string `gorm:"column:id"`
			Name   string `gorm:"column:name"`
			Status string `gorm:"column:status"`
		}
		var prs []ProdRow
		if err := tx.WithContext(ctx).
			Table("products").
			Select("id, name, status").
			Where("id IN ?", productIDs).
			Find(&prs).Error; err != nil {
			return err
		}
		pmap := map[string]ProdRow{}
		for _, p := range prs {
			pmap[p.ID] = p
		}
		for _, v := range vs {
			p, ok := pmap[v.ProductID]
			if !ok || p.Status != "active" {
				return ErrProductUnavailable
			}
		}

		// 6) totals + order_items prepare
		subtotal := 0
		oi := make([]OrderItem, 0, len(ids))

		for _, vid := range ids {
			v := vmap[vid]
			q := want[vid]
			line := v.PriceCents * q
			subtotal += line

			p := pmap[v.ProductID]

			oi = append(oi, OrderItem{
				ID:             uuid.NewString(),
				OrderID:        "", // aşağıda set
				VariantID:      v.ID,
				ProductName:    p.Name,
				SKU:            v.SKU,
				OptionsJSON:    v.Options,
				UnitPriceCents: v.PriceCents,
				Currency:       currency,
				Quantity:       q,
				LineTotalCents: line,
				CreatedAt:      now,
			})
		}

		total := subtotal + in.TaxCents + in.ShippingCents - in.DiscountCents
		if total < 0 {
			total = 0
		}

		// 7) orders insert
		orderID := uuid.NewString()
		o := Order{
			ID:         orderID,
			UserID:     in.UserID,
			GuestEmail: in.GuestEmail,
			Status:     "created",
			Currency:   currency,

			SubtotalCents: subtotal,
			TaxCents:      in.TaxCents,
			ShippingCents: in.ShippingCents,
			DiscountCents: in.DiscountCents,
			TotalCents:    total,

			ShippingAddressJSON: in.ShippingAddressJSON,
			BillingAddressJSON:  in.BillingAddressJSON,

			IdempotencyKey: in.IdempotencyKey,

			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := tx.WithContext(ctx).Create(&o).Error; err != nil {
			// idempotency unique çakışması: mevcut order'ı dön
			if isDuplicateKey(err) && in.UserID != nil && in.IdempotencyKey != nil && *in.IdempotencyKey != "" {
				var existing Order
				if err2 := tx.WithContext(ctx).First(&existing, "user_id = ? AND idempotency_key = ?", *in.UserID, *in.IdempotencyKey).Error; err2 == nil {
					out = CreateFromCartResult{
						OrderID:       existing.ID,
						Status:        existing.Status,
						Currency:      existing.Currency,
						TotalCents:    existing.TotalCents,
						SubtotalCents: existing.SubtotalCents,
						Idempotent:    true,
					}
					return nil
				}
			}
			return err
		}

		// 8) order_items insert
		for i := range oi {
			oi[i].OrderID = orderID
		}
		if err := tx.WithContext(ctx).Create(&oi).Error; err != nil {
			return err
		}

		// 9) cart cleanup
		if in.UserID != nil {
			// user cart: cart kalsın, items silinsin
			if err := tx.WithContext(ctx).Exec("DELETE FROM cart_items WHERE cart_id = ?", in.CartID).Error; err != nil {
				return err
			}
		} else {
			// guest cart: cart silinsin (cascade ile items gider)
			if err := tx.WithContext(ctx).Exec("DELETE FROM carts WHERE id = ?", in.CartID).Error; err != nil {
				return err
			}
		}

		out = CreateFromCartResult{
			OrderID:       orderID,
			Status:        o.Status,
			Currency:      currency,
			TotalCents:    total,
			SubtotalCents: subtotal,
			Idempotent:    false,
		}
		return nil
	})

	return out, err
}

func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	return false
}
