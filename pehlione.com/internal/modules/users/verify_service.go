package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"pehlione.com/app/internal/modules/email"
)

type EmailVerification struct {
	ID        string     `gorm:"primaryKey;column:id"`
	UserID    string     `gorm:"column:user_id"`
	CodeHash  []byte     `gorm:"column:code_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (EmailVerification) TableName() string { return "email_verifications" }

type VerifyService struct {
	db         *gorm.DB
	emailSvc   *email.OutboxService
	appBaseURL string
	fromName   string
}

func NewVerifyService(db *gorm.DB, emailSvc *email.OutboxService, appBaseURL, fromName string) *VerifyService {
	return &VerifyService{
		db:         db,
		emailSvc:   emailSvc,
		appBaseURL: appBaseURL,
		fromName:   fromName,
	}
}

func (s *VerifyService) StartEmailVerification(ctx context.Context, userID, userEmail string) error {
	if s.emailSvc == nil {
		log.Printf("verify_service: emailSvc is nil, skipping verification email")
		return nil
	}

	log.Printf("verify_service: starting email verification for user=%s email=%s", userID, userEmail)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rawToken, err := randomToken(32)
		if err != nil {
			log.Printf("verify_service: failed to generate token: %v", err)
			return err
		}
		hash := sha256.Sum256([]byte(rawToken))

		// Delete any existing verification records for this user
		_ = tx.WithContext(ctx).Where("user_id = ?", userID).Delete(&EmailVerification{}).Error

		// Create new email verification record
		id := generateID() // Generate a UUID
		ev := EmailVerification{
			ID:        id,
			UserID:    userID,
			CodeHash:  hash[:],
			ExpiresAt: time.Now().Add(30 * time.Minute),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := tx.WithContext(ctx).Create(&ev).Error; err != nil {
			log.Printf("verify_service: failed to create email verification record: %v", err)
			return err
		}

		verifyURL := strings.TrimRight(s.appBaseURL, "/") + "/verify-email?token=" + rawToken
		log.Printf("verify_service: enqueuing email to %s with verify_url=%s", userEmail, verifyURL)
		err = s.emailSvc.EnqueueTx(ctx, tx, email.Job{
			To:       userEmail,
			Template: "verify_email",
			Payload: map[string]any{
				"VerifyURL": verifyURL,
				"FromName":  s.fromName,
			},
		})
		if err != nil {
			log.Printf("verify_service: failed to enqueue email: %v", err)
		} else {
			log.Printf("verify_service: email successfully enqueued")
		}
		return err
	})
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func generateID() string {
	return uuid.NewString()
}
