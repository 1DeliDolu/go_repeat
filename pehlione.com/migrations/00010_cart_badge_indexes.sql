-- +goose Up

-- Optimize cart badge query: user_id, status, updated_at for efficient subquery
ALTER TABLE carts
  ADD INDEX ix_carts_user_status_updated (user_id, status, updated_at DESC);

-- +goose Down
ALTER TABLE carts
  DROP INDEX ix_carts_user_status_updated;
