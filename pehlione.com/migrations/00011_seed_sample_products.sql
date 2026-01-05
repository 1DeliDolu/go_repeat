-- +goose Up
use pehlione_go;

-- Insert sample products
INSERT INTO products (id, name, slug, description, status, created_at, updated_at)
VALUES
  ('prod_001', 'Sample Product 1', 'sample-product-1', 'A high-quality product for your needs.', 'active', NOW(3), NOW(3)),
  ('prod_002', 'Sample Product 2', 'sample-product-2', 'Premium quality with great features.', 'active', NOW(3), NOW(3)),
  ('prod_003', 'Sample Product 3', 'sample-product-3', 'Exclusive offer with limited quantity.', 'active', NOW(3), NOW(3));

-- Insert sample variants
INSERT INTO product_variants (id, product_id, sku, options_json, price_cents, currency, stock, created_at, updated_at)
VALUES
  ('var_001', 'prod_001', 'SKU-001', '{}', 1999, 'USD', 100, NOW(3), NOW(3)),
  ('var_002', 'prod_002', 'SKU-002', '{}', 2999, 'USD', 75, NOW(3), NOW(3)),
  ('var_003', 'prod_003', 'SKU-003', '{}', 3999, 'USD', 50, NOW(3), NOW(3));

-- +goose Down
use pehlione_go;

DELETE FROM product_variants WHERE id IN ('var_001', 'var_002', 'var_003');
DELETE FROM products WHERE id IN ('prod_001', 'prod_002', 'prod_003', 'prod_004');
