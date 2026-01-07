-- Add name and address fields to users table
ALTER TABLE users ADD COLUMN first_name VARCHAR(255) NULL;
ALTER TABLE users ADD COLUMN last_name VARCHAR(255) NULL;
ALTER TABLE users ADD COLUMN address TEXT NULL;
