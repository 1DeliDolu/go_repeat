package checkout

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StockLine struct {
	VariantID string
	Qty       int
}

// DeductStockInTx: DIŞARIDAN verilen tx içinde çalışır (nested tx yok).
// Order creation tx'inin içinden çağrılır.
func DeductStockInTx(ctx context.Context, tx *gorm.DB, lines []StockLine) error {
	if len(lines) == 0 {
		return nil
	}

	// deterministik sıra
	sort.Slice(lines, func(i, j int) bool { return lines[i].VariantID < lines[j].VariantID })

	want := make(map[string]int, len(lines))
	for _, ln := range lines {
		q := ln.Qty
		if q < 1 {
			q = 1
		}
		want[ln.VariantID] += q
	}

	ids := make([]string, 0, len(want))
	for id := range want {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	type VariantRow struct {
		ID    string `gorm:"column:id"`
		Stock int    `gorm:"column:stock"`
	}
	var rows []VariantRow

	// SELECT ... FOR UPDATE
	if err := tx.WithContext(ctx).
		Table("product_variants").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return err
	}

	avail := make(map[string]int, len(rows))
	for _, r := range rows {
		avail[r.ID] = r.Stock
	}

	var oos []OutOfStockItem
	for _, id := range ids {
		req := want[id]
		av, ok := avail[id]
		if !ok || av < req {
			oos = append(oos, OutOfStockItem{VariantID: id, Requested: req, Available: av})
		}
	}
	if len(oos) > 0 {
		return &OutOfStockError{Items: oos}
	}

	// stock = stock - qty
	for _, id := range ids {
		req := want[id]
		res := tx.WithContext(ctx).
			Table("product_variants").
			Where("id = ?", id).
			UpdateColumn("stock", gorm.Expr("stock - ?", req))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return &OutOfStockError{Items: []OutOfStockItem{{VariantID: id, Requested: req, Available: 0}}}
		}
	}

	return nil
}

// DeductStockTx: wrapper (retry + tx) — dışarıdan çağıranlar için.
func DeductStockTx(ctx context.Context, db *gorm.DB, lines []StockLine) error {
	return withTxRetry(ctx, db, 3, func(tx *gorm.DB) error {
		return DeductStockInTx(ctx, tx, lines)
	})
}

// --- retry helpers (deadlock/lock timeout) ---

func withTxRetry(ctx context.Context, db *gorm.DB, attempts int, fn func(tx *gorm.DB) error) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error

	for i := 0; i < attempts; i++ {
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return fn(tx)
		})
		if err == nil {
			return nil
		}
		lastErr = err

		if isRetryableMySQLError(err) && i < attempts-1 {
			// küçük backoff
			time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)
			continue
		}
		return err
	}
	return lastErr
}

func isRetryableMySQLError(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		// 1213: Deadlock found; 1205: Lock wait timeout
		return me.Number == 1213 || me.Number == 1205
	}
	return false
}
