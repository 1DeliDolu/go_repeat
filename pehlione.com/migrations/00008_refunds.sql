-- +goose Up
ALTER TABLE orders
  ADD COLUMN refunded_cents INT NOT NULL DEFAULT 0 AFTER paid_at,
  ADD COLUMN refunded_at DATETIME(3) NULL AFTER refunded_cents;

CREATE TABLE refunds (
  id CHAR(36) NOT NULL,
  order_id CHAR(36) NOT NULL,
  payment_id CHAR(36) NOT NULL,

  provider VARCHAR(64) NOT NULL,
  provider_ref VARCHAR(128) NULL,

  status VARCHAR(32) NOT NULL, -- initiated|succeeded|failed
  amount_cents INT NOT NULL,
  currency CHAR(3) NOT NULL,

  idempotency_key VARCHAR(64) NOT NULL,
  reason VARCHAR(255) NULL,
  error_message VARCHAR(255) NULL,

  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

  PRIMARY KEY (id),
  UNIQUE KEY ux_refunds_payment_idem (payment_id, idempotency_key),
  KEY ix_refunds_order_id (order_id),
  KEY ix_refunds_payment_id (payment_id),
  KEY ix_refunds_status_created (status, created_at),

  CONSTRAINT fk_refunds_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
  CONSTRAINT fk_refunds_payment FOREIGN KEY (payment_id) REFERENCES payments(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_financial_entries (
  id CHAR(36) NOT NULL,
  order_id CHAR(36) NOT NULL,

  event VARCHAR(32) NOT NULL,     -- payment_succeeded|refund_succeeded|refund_failed
  amount_cents INT NOT NULL,      -- +in, -out
  currency CHAR(3) NOT NULL,

  ref_type VARCHAR(16) NOT NULL,  -- payment|refund
  ref_id CHAR(36) NOT NULL,

  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

  PRIMARY KEY (id),
  KEY ix_order_fin_entries_order_created (order_id, created_at),
  KEY ix_order_fin_entries_ref (ref_type, ref_id),

  CONSTRAINT fk_order_fin_entries_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS order_financial_entries;
DROP TABLE IF EXISTS refunds;
ALTER TABLE orders DROP COLUMN refunded_at;
ALTER TABLE orders DROP COLUMN refunded_cents;
