package main

import (
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

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get DB: %v", err)
	}

	sql := `
	CREATE TABLE IF NOT EXISTS order_items (
	  id CHAR(36) NOT NULL,
	  order_id CHAR(36) NOT NULL,
	  variant_id CHAR(36) NOT NULL,
	  product_name VARCHAR(255) NOT NULL,
	  sku VARCHAR(64) NOT NULL,
	  options_json JSON NOT NULL,
	  unit_price_cents INT NOT NULL,
	  currency CHAR(3) NOT NULL DEFAULT 'EUR',
	  quantity INT NOT NULL,
	  line_total_cents INT NOT NULL,
	  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	  PRIMARY KEY (id),
	  KEY ix_order_items_order_id (order_id),
	  KEY ix_order_items_variant_id (variant_id),
	  CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
	  CONSTRAINT fk_order_items_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE RESTRICT
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

	CREATE TABLE IF NOT EXISTS order_financial_entries (
	  id CHAR(36) NOT NULL,
	  order_id CHAR(36) NOT NULL,
	  event VARCHAR(32) NOT NULL,
	  amount_cents INT NOT NULL,
	  currency CHAR(3) NOT NULL,
	  ref_type VARCHAR(16) NOT NULL,
	  ref_id CHAR(36) NOT NULL,
	  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	  PRIMARY KEY (id),
	  KEY ix_order_fin_entries_order_created (order_id, created_at),
	  KEY ix_order_fin_entries_ref (ref_type, ref_id),
	  CONSTRAINT fk_order_fin_entries_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	if _, err := sqlDB.Exec(sql); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	log.Println("✓ order_items table created successfully")
	log.Println("✓ order_financial_entries table created successfully")
}
