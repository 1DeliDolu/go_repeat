package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "musta:pehlione@tcp(localhost:3306)/pehlione_go?parseTime=true&multiStatements=true&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Add columns to orders table
	addCol := func(sql string) {
		if err := db.Exec(sql).Error; err != nil {
			if err.Error() != "Error 1060 (42S21): Duplicate column name 'base_currency'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_subtotal_cents'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_tax_cents'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_shipping_cents'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_discount_cents'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_total_cents'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'display_currency'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'fx_rate'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'fx_source'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_unit_price_cents'" &&
				err.Error() != "Error 1060 (42S21): Duplicate column name 'base_line_total_cents'" {
				log.Fatalf("Failed: %v", err)
			}
		}
	}

	addCol(`ALTER TABLE orders ADD COLUMN base_currency CHAR(3) NOT NULL DEFAULT 'TRY' AFTER currency`)
	addCol(`ALTER TABLE orders ADD COLUMN base_subtotal_cents INT NOT NULL DEFAULT 0 AFTER base_currency`)
	addCol(`ALTER TABLE orders ADD COLUMN base_tax_cents INT NOT NULL DEFAULT 0 AFTER base_subtotal_cents`)
	addCol(`ALTER TABLE orders ADD COLUMN base_shipping_cents INT NOT NULL DEFAULT 0 AFTER base_tax_cents`)
	addCol(`ALTER TABLE orders ADD COLUMN base_discount_cents INT NOT NULL DEFAULT 0 AFTER base_shipping_cents`)
	addCol(`ALTER TABLE orders ADD COLUMN base_total_cents INT NOT NULL DEFAULT 0 AFTER base_discount_cents`)
	addCol(`ALTER TABLE orders ADD COLUMN display_currency CHAR(3) NOT NULL DEFAULT 'TRY' AFTER base_total_cents`)
	addCol(`ALTER TABLE orders ADD COLUMN fx_rate DECIMAL(18,8) NULL AFTER display_currency`)
	addCol(`ALTER TABLE orders ADD COLUMN fx_source VARCHAR(32) NULL AFTER fx_rate`)
	addCol(`ALTER TABLE order_items ADD COLUMN base_currency CHAR(3) NOT NULL DEFAULT 'TRY' AFTER currency`)
	addCol(`ALTER TABLE order_items ADD COLUMN base_unit_price_cents INT NOT NULL DEFAULT 0 AFTER base_currency`)
	addCol(`ALTER TABLE order_items ADD COLUMN base_line_total_cents INT NOT NULL DEFAULT 0 AFTER base_unit_price_cents`)

	fmt.Println("✓ Multi-currency columns added successfully!")
}
