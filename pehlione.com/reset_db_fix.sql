-- Reset database script - drop all tables and goose metadata
DROP TABLE IF EXISTS "order_financial_entries";
DROP TABLE IF EXISTS "order_events";
DROP TABLE IF EXISTS "order_items";
DROP TABLE IF EXISTS "payments";
DROP TABLE IF EXISTS "orders";
DROP TABLE IF EXISTS "provider_events";
DROP TABLE IF EXISTS "refunds";
DROP TABLE IF EXISTS "cart_items";
DROP TABLE IF EXISTS "carts";
DROP TABLE IF EXISTS "product_images";
DROP TABLE IF EXISTS "product_variants";
DROP TABLE IF EXISTS "products";
DROP TABLE IF EXISTS "sessions";
DROP TABLE IF EXISTS "users";
DROP TABLE IF EXISTS "goose_db_version";
