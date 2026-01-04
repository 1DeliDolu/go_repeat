-- +goose Up
CREATE TABLE provider_events (
  id CHAR(36) NOT NULL,
  provider VARCHAR(64) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  event_type VARCHAR(64) NOT NULL,

  payload_json JSON NOT NULL,

  received_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  processed_at DATETIME(3) NULL,
  process_error VARCHAR(255) NULL,

  PRIMARY KEY (id),
  UNIQUE KEY ux_provider_events_provider_event (provider, event_id),
  KEY ix_provider_events_provider_received (provider, received_at),
  KEY ix_provider_events_processed (processed_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS provider_events;
