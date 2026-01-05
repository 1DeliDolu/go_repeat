-- +goose Up
use pehlione_go;

-- Insert seed admin user
-- Email: delione@pehlione.com, Password: password123
-- Email: deli@pehlione.com, Password: password123
INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
VALUES
  ('admin_001', 'delione@pehlione.com', '$2a$10$YUMU3rFyD2Wm51wPQppTMed3DbyavEdPZ5R9aTfM3Zihfe8uk9D/W', 'admin', NOW(3), NOW(3)),
  ('user_001', 'deli@pehlione.com', '$2a$10$YUMU3rFyD2Wm51wPQppTMed3DbyavEdPZ5R9aTfM3Zihfe8uk9D/W', 'user', NOW(3), NOW(3));

-- +goose Down
use pehlione_go;

DELETE FROM users WHERE id IN ('admin_001', 'user_001');
