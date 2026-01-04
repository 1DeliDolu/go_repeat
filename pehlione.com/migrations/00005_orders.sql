-- +goose Up
use pehlione_go;
CREATE TABLE orders (
  id CHAR(36) NOT NULL,

  user_id CHAR(36) NULL,
  guest_email VARCHAR(255) NULL,

  status VARCHAR(32) NOT NULL DEFAULT 'created',
  currency CHAR(3) NOT NULL DEFAULT 'EUR',

  subtotal_cents INT NOT NULL DEFAULT 0,
  tax_cents INT NOT NULL DEFAULT 0,
  shipping_cents INT NOT NULL DEFAULT 0,
  discount_cents INT NOT NULL DEFAULT 0,
  total_cents INT NOT NULL DEFAULT 0,

  shipping_address_json JSON NULL,
  billing_address_json JSON NULL,

  idempotency_key VARCHAR(64) NULL,

  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

  PRIMARY KEY (id),

  KEY ix_orders_user_id_created_at (user_id, created_at),
  KEY ix_orders_status_created_at (status, created_at),

  UNIQUE KEY ux_orders_user_id_idempotency (user_id, idempotency_key),

  CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_items (
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

-- +goose Down
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
