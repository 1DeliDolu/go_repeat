-- +goose Up
USE pehlione_go;
ALTER TABLE product_images
  ADD COLUMN storage_key VARCHAR(1024) NULL AFTER product_id;

-- mevcut kayıt varsa url’i key gibi kullan
UPDATE product_images
  SET storage_key = url
  WHERE storage_key IS NULL;

ALTER TABLE product_images
  MODIFY storage_key VARCHAR(1024) NOT NULL;

-- +goose Down
ALTER TABLE product_images
  DROP COLUMN storage_key;
