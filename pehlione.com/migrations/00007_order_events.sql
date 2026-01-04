-- +goose Up
CREATE TABLE order_events (
  id CHAR(36) NOT NULL,
  order_id CHAR(36) NOT NULL,
  actor_user_id CHAR(36) NOT NULL,

  action VARCHAR(32) NOT NULL,          -- ship|deliver|cancel|refund|note
  from_status VARCHAR(32) NOT NULL,
  to_status VARCHAR(32) NOT NULL,

  note VARCHAR(255) NULL,

  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

  PRIMARY KEY (id),
  KEY ix_order_events_order_id_created_at (order_id, created_at),
  KEY ix_order_events_actor_created_at (actor_user_id, created_at),

  CONSTRAINT fk_order_events_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
  CONSTRAINT fk_order_events_actor FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS order_events;
