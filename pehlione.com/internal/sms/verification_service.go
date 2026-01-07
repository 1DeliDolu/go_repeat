package sms

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type SMSVerification struct {
	ID          int64      `gorm:"primaryKey"`
	UserID      string     `gorm:"column:user_id"`
	PhoneE164   string     `gorm:"column:phone_e164"`
	CodeHash    string     `gorm:"column:code_hash"`
	Attempts    int        `gorm:"column:attempts"`
	MaxAttempts int        `gorm:"column:max_attempts"`
	ExpiresAt   time.Time  `gorm:"column:expires_at"`
	VerifiedAt  *time.Time `gorm:"column:verified_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (SMSVerification) TableName() string { return "sms_verifications" }

type RateLimit struct {
	ID            int64     `gorm:"primaryKey"`
	UserID        string    `gorm:"column:user_id"`
	Action        string    `gorm:"column:action"`
	PhoneE164     *string   `gorm:"column:phone_e164"`
	AttemptCount  int       `gorm:"column:attempt_count"`
	LastAttemptAt time.Time `gorm:"column:last_attempt_at"`
	ExpiresAt     time.Time `gorm:"column:expires_at"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (RateLimit) TableName() string { return "sms_rate_limits" }

type SentLog struct {
	ID                int64      `gorm:"primaryKey"`
	UserID            string     `gorm:"column:user_id"`
	PhoneE164         string     `gorm:"column:phone_e164"`
	MessageType       string     `gorm:"column:message_type"`
	Status            string     `gorm:"column:status"`
	ProviderMessageID *string    `gorm:"column:provider_message_id"`
	ErrorMessage      *string    `gorm:"column:error_message"`
	SentAt            *time.Time `gorm:"column:sent_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
}

func (SentLog) TableName() string { return "sms_sent_logs" }

type VerificationService struct {
	db       *gorm.DB
	provider SMSProvider
}

func NewVerificationService(db *gorm.DB, provider SMSProvider) *VerificationService {
	return &VerificationService{db: db, provider: provider}
}

// GenerateOTP generates a 6-digit OTP code
func (s *VerificationService) GenerateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Convert 3 random bytes to 6-digit number (0-999999)
	num := int((int(b[0])<<16)|(int(b[1])<<8)|int(b[2])) % 1000000
	return fmt.Sprintf("%06d", num), nil
}

// StartPhoneVerification creates a verification code and sends SMS
func (s *VerificationService) StartPhoneVerification(ctx context.Context, userID, phoneE164 string) (string, error) {
	// Check rate limit
	if blocked, err := s.isRateLimited(ctx, userID, "verify_phone"); blocked {
		return "", fmt.Errorf("too many verification attempts, please try again later")
	} else if err != nil {
		return "", err
	}

	// Generate OTP
	code, err := s.GenerateOTP()
	if err != nil {
		return "", err
	}

	// Hash the code
	hash := sha256.Sum256([]byte(code))
	hashHex := hex.EncodeToString(hash[:])

	// Delete previous unverified codes
	_ = s.db.WithContext(ctx).Where("user_id = ? AND verified_at IS NULL", userID).Delete(&SMSVerification{}).Error

	// Create verification record
	verification := SMSVerification{
		UserID:      userID,
		PhoneE164:   phoneE164,
		CodeHash:    hashHex,
		Attempts:    0,
		MaxAttempts: 3,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		CreatedAt:   time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&verification).Error; err != nil {
		return "", err
	}

	// Send SMS
	idempotencyKey := fmt.Sprintf("%s-%s-%d", userID, "verify_phone", time.Now().Unix())
	providerMsgID, err := s.provider.Send(ctx, phoneE164, fmt.Sprintf("Doğrulama kodu: %s (10 dakika geçerli)", code), idempotencyKey)

	// Log the SMS
	logEntry := SentLog{
		UserID:      userID,
		PhoneE164:   phoneE164,
		MessageType: "phone_verification",
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err != nil {
		logEntry.Status = "failed"
		errMsg := err.Error()
		logEntry.ErrorMessage = &errMsg
	} else {
		logEntry.Status = "sent"
		logEntry.ProviderMessageID = &providerMsgID
		sentAt := time.Now()
		logEntry.SentAt = &sentAt
	}

	_ = s.db.WithContext(ctx).Create(&logEntry).Error

	// Update rate limit
	s.recordAttempt(ctx, userID, "verify_phone", &phoneE164)

	return code, err
}

// VerifyPhoneCode verifies the OTP code
func (s *VerificationService) VerifyPhoneCode(ctx context.Context, userID, code string) (phoneE164 string, err error) {
	// Hash the code
	hash := sha256.Sum256([]byte(code))
	hashHex := hex.EncodeToString(hash[:])

	// Find the verification record
	var verification SMSVerification
	if err := s.db.WithContext(ctx).Where(
		"user_id = ? AND code_hash = ? AND expires_at > ? AND verified_at IS NULL",
		userID, hashHex, time.Now(),
	).First(&verification).Error; err != nil {
		// Increment attempts (best effort)
		_ = s.db.WithContext(ctx).Model(&SMSVerification{}).Where(
			"user_id = ? AND code_hash = ? AND expires_at > ?",
			userID, hashHex, time.Now(),
		).Update("attempts", gorm.Expr("attempts + 1")).Error

		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("invalid or expired verification code")
		}
		return "", err
	}

	// Check max attempts
	if verification.Attempts >= verification.MaxAttempts {
		return "", fmt.Errorf("too many attempts, please request a new code")
	}

	// Mark as verified
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&verification).Update("verified_at", now).Error; err != nil {
		return "", err
	}

	return verification.PhoneE164, nil
}

// isRateLimited checks if user has exceeded rate limit
func (s *VerificationService) isRateLimited(ctx context.Context, userID, action string) (bool, error) {
	var limit RateLimit
	err := s.db.WithContext(ctx).Where(
		"user_id = ? AND action = ? AND expires_at > ?",
		userID, action, time.Now(),
	).First(&limit).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 3 attempts in 5 minutes = rate limited
	return limit.AttemptCount >= 3, nil
}

// recordAttempt records a verification attempt for rate limiting
func (s *VerificationService) recordAttempt(ctx context.Context, userID, action string, phoneE164 *string) {
	now := time.Now()
	expiresAt := now.Add(5 * time.Minute)

	// Try to update existing rate limit
	result := s.db.WithContext(ctx).Model(&RateLimit{}).Where(
		"user_id = ? AND action = ?", userID, action,
	).Updates(map[string]interface{}{
		"attempt_count":   gorm.Expr("attempt_count + 1"),
		"last_attempt_at": now,
		"expires_at":      expiresAt,
	})

	// If no rows affected, create new rate limit
	if result.RowsAffected == 0 {
		s.db.WithContext(ctx).Create(&RateLimit{
			UserID:        userID,
			Action:        action,
			PhoneE164:     phoneE164,
			AttemptCount:  1,
			LastAttemptAt: now,
			ExpiresAt:     expiresAt,
			CreatedAt:     now,
		})
	}
}

// GetVerificationHistory returns recent SMS verifications
func (s *VerificationService) GetVerificationHistory(ctx context.Context, userID string, limit int) ([]SentLog, error) {
	var logs []SentLog
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
