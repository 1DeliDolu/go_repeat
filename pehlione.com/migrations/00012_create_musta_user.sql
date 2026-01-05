-- +goose Up
-- Create user for TCP connections (localhost, 127.0.0.1, and all hosts)
CREATE USER IF NOT EXISTS 'musta'@'localhost' IDENTIFIED BY 'pehlione';
CREATE USER IF NOT EXISTS 'musta'@'127.0.0.1' IDENTIFIED BY 'pehlione';
CREATE USER IF NOT EXISTS 'musta'@'%' IDENTIFIED BY 'pehlione';

-- Grant privileges
GRANT ALL PRIVILEGES ON pehlione_go.* TO 'musta'@'localhost';
GRANT ALL PRIVILEGES ON pehlione_go.* TO 'musta'@'127.0.0.1';
GRANT ALL PRIVILEGES ON pehlione_go.* TO 'musta'@'%';
FLUSH PRIVILEGES;

-- +goose Down
DROP USER IF EXISTS 'musta'@'localhost';
DROP USER IF EXISTS 'musta'@'127.0.0.1';
DROP USER IF EXISTS 'musta'@'%';
