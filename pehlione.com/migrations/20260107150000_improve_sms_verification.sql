-- SMS verification codes table for OTP tracking
CREATE TABLE sms_verifications (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  phone_e164 VARCHAR(32) NOT NULL,
  code_hash VARCHAR(64) NOT NULL,
  attempts INT DEFAULT 0,
  max_attempts INT DEFAULT 3,
  expires_at DATETIME(3) NOT NULL,
  verified_at DATETIME(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_user_id (user_id),
  KEY idx_expires_at (expires_at),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- SMS rate limiting table
CREATE TABLE sms_rate_limits (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  action VARCHAR(32) NOT NULL,
  phone_e164 VARCHAR(32),
  attempt_count INT DEFAULT 1,
  last_attempt_at DATETIME(3) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_user_action (user_id, action),
  KEY idx_expires_at (expires_at),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- SMS sent log table
CREATE TABLE sms_sent_logs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  phone_e164 VARCHAR(32) NOT NULL,
  message_type VARCHAR(32) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  provider_message_id VARCHAR(255),
  error_message TEXT,
  sent_at DATETIME(3),
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_user_id (user_id),
  KEY idx_phone (phone_e164),
  KEY idx_status (status),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
