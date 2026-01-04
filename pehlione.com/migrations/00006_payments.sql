-- +goose Up
ALTER TABLE orders
  ADD COLUMN paid_at DATETIME(3) NULL AFTER status;

CREATE TABLE payments (
  id CHAR(36) NOT NULL,
  order_id CHAR(36) NOT NULL,

  provider VARCHAR(64) NOT NULL,
  provider_ref VARCHAR(128) NULL,

  status VARCHAR(32) NOT NULL,
  amount_cents INT NOT NULL,
  currency CHAR(3) NOT NULL,

  idempotency_key VARCHAR(64) NOT NULL,
  error_message VARCHAR(255) NULL,

  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

  PRIMARY KEY (id),
  UNIQUE KEY ux_payments_order_idem (order_id, idempotency_key),
  KEY ix_payments_order_id (order_id),
  KEY ix_payments_status_created (status, created_at),

  CONSTRAINT fk_payments_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS payments;
ALTER TABLE orders DROP COLUMN paid_at;
