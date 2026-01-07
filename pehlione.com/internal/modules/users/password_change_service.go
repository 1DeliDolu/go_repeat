package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"gorm.io/gorm"
	"pehlione.com/app/internal/modules/email"
)

type PasswordChange struct {
	ID              int64      `gorm:"primaryKey"`
	UserID          string     `gorm:"column:user_id"`
	TokenHash       string     `gorm:"column:token_hash"`
	NewPasswordHash string     `gorm:"column:new_password_hash"`
	ExpiresAt       time.Time  `gorm:"column:expires_at"`
	UsedAt          *time.Time `gorm:"column:used_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (PasswordChange) TableName() string { return "password_changes" }

type PasswordChangeService struct {
	db         *gorm.DB
	emailSvc   *email.OutboxService
	appBaseURL string
	fromName   string
}

func NewPasswordChangeService(db *gorm.DB, emailSvc *email.OutboxService, appBaseURL, fromName string) *PasswordChangeService {
	return &PasswordChangeService{
		db:         db,
		emailSvc:   emailSvc,
		appBaseURL: appBaseURL,
		fromName:   fromName,
	}
}

func (s *PasswordChangeService) StartPasswordChange(ctx context.Context, userID, userEmail, newPasswordHash string) error {
	if s.emailSvc == nil {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Generate token
		rawToken, err := randomToken(32)
		if err != nil {
			return err
		}
		hash := sha256.Sum256([]byte(rawToken))
		hashHex := hex.EncodeToString(hash[:])

		// Delete any existing pending password changes for this user
		_ = tx.WithContext(ctx).Where("user_id = ? AND used_at IS NULL", userID).Delete(&PasswordChange{}).Error

		// Create new password change record
		pc := PasswordChange{
			UserID:          userID,
			TokenHash:       hashHex,
			NewPasswordHash: newPasswordHash,
			ExpiresAt:       time.Now().Add(24 * time.Hour),
			CreatedAt:       time.Now(),
		}
		if err := tx.WithContext(ctx).Create(&pc).Error; err != nil {
			return err
		}

		// Send confirmation email
		confirmURL := strings.TrimRight(s.appBaseURL, "/") + "/confirm-password-change?token=" + rawToken
		cancelURL := strings.TrimRight(s.appBaseURL, "/") + "/cancel-password-change?token=" + rawToken

		return s.emailSvc.EnqueueTx(ctx, tx, email.Job{
			To:       userEmail,
			Template: "password_change_confirmation",
			Payload: map[string]any{
				"ConfirmURL": confirmURL,
				"CancelURL":  cancelURL,
				"FromName":   s.fromName,
				"ExpiresIn":  "24 hours",
			},
		})
	})
}

func (s *PasswordChangeService) ConfirmPasswordChange(ctx context.Context, tokenHash string) (userID string, newPasswordHash string, err error) {
	var pc PasswordChange
	if err := s.db.WithContext(ctx).Where("token_hash = ? AND expires_at > ? AND used_at IS NULL", tokenHash, time.Now()).First(&pc).Error; err != nil {
		return "", "", err
	}

	return pc.UserID, pc.NewPasswordHash, nil
}

func (s *PasswordChangeService) ApplyPasswordChange(ctx context.Context, tokenHash string, userID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Mark as used
		if err := tx.WithContext(ctx).Model(&PasswordChange{}).Where("token_hash = ?", tokenHash).Update("used_at", time.Now()).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *PasswordChangeService) CancelPasswordChange(ctx context.Context, tokenHash string) error {
	return s.db.WithContext(ctx).Model(&PasswordChange{}).Where("token_hash = ?", tokenHash).Update("used_at", time.Now()).Error
}
