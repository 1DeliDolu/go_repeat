package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"pehlione.com/app/internal/http/flash"
	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/internal/modules/auth"
	"pehlione.com/app/internal/sms"
	"pehlione.com/app/pkg/view"
)

type SmsHandler struct {
	db              *gorm.DB
	smsRepo         *sms.OutboxRepository
	logger          *slog.Logger
	flash           *flash.Codec
	verificationSvc *sms.VerificationService
}

func NewSmsHandler(db *gorm.DB, smsRepo *sms.OutboxRepository, flashCodec *flash.Codec, logger *slog.Logger) *SmsHandler {
	return &SmsHandler{
		db:      db,
		smsRepo: smsRepo,
		flash:   flashCodec,
		logger:  logger,
	}
}

func (h *SmsHandler) SetVerificationService(svc *sms.VerificationService) {
	h.verificationSvc = svc
}

func (h *SmsHandler) PostAccountSMS(c *gin.Context) {
	// 1. Get user from context
	ctxUser, ok := middleware.CurrentUser(c)
	if !ok {
		render.Redirect(c, "/login")
		return
	}
	var currentUser auth.User
	if err := h.db.First(&currentUser, "id = ?", ctxUser.ID).Error; err != nil {
		render.Redirect(c, "/login")
		return
	}

	// 2. Parse form
	phone := c.PostForm("phone")
	optIn := c.PostForm("sms_opt_in") == "on"

	// 3. Basic validation
	if phone == "" {
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Phone number is required.")
		return
	}

	// 4. Update user
	updates := map[string]interface{}{
		"phone_e164": &phone,
		"sms_opt_in": optIn,
	}
	if !optIn {
		updates["sms_opt_out_at"] = time.Now()
	}
	err := h.db.WithContext(c.Request.Context()).Model(&currentUser).Updates(updates).Error

	if err != nil {
		h.logger.Error("failed to update user phone", "error", err, "user_id", currentUser.ID)
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Failed to update your settings.")
		return
	}

	// 5. Create consent record
	action := "opt_out"
	if optIn {
		action = "opt_in"
	}
	consent := sms.Consent{
		UserID:    currentUser.ID,
		PhoneE164: phone,
		Action:    action,
		Source:    "profile",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		CreatedAt: time.Now(),
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&consent).Error; err != nil {
		h.logger.Error("failed to create consent record", "error", err, "user_id", currentUser.ID)
		// Do not block the user for this, just log it.
	}

	render.RedirectWithFlash(c, h.flash, "/account", view.FlashSuccess, "Your SMS settings have been updated.")
}

func (h *SmsHandler) PostAccountSMSVerify(c *gin.Context) {
	// 1. Get user from context
	ctxUser, ok := middleware.CurrentUser(c)
	if !ok {
		render.Redirect(c, "/login")
		return
	}
	var currentUser auth.User
	if err := h.db.First(&currentUser, "id = ?", ctxUser.ID).Error; err != nil {
		render.Redirect(c, "/login")
		return
	}

	// 2. Parse form
	code := c.PostForm("code")

	// 3. Basic validation
	if code == "" {
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama kodu gereklidir.")
		return
	}

	// 4. Use verification service if available
	if h.verificationSvc != nil {
		phoneE164, err := h.verificationSvc.VerifyPhoneCode(c.Request.Context(), currentUser.ID, code)
		if err != nil {
			h.logger.Error("failed to verify phone code", "error", err, "user_id", currentUser.ID)
			render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama başarısız: "+err.Error())
			return
		}

		// Update user phone verified timestamp
		now := time.Now()
		if err := h.db.WithContext(c.Request.Context()).Model(&currentUser).Updates(map[string]interface{}{
			"phone_e164":        phoneE164,
			"phone_verified_at": now,
		}).Error; err != nil {
			h.logger.Error("failed to update user phone_verified_at", "error", err, "user_id", currentUser.ID)
			render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama sırasında bir hata oluştu.")
			return
		}

		render.RedirectWithFlash(c, h.flash, "/account", view.FlashSuccess, "Telefon numaranız başarıyla doğrulandı.")
		return
	}

	// Fallback: Manual verification
	var verification sms.PhoneVerification
	err := h.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND used_at IS NULL", currentUser.ID).
		Order("created_at DESC").
		First(&verification).Error

	if err != nil {
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Geçersiz doğrulama kodu.")
		return
	}

	// Check expiry
	if time.Now().After(verification.ExpiresAt) {
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama kodunun süresi doldu.")
		return
	}

	// Check code hash
	hasher := sha256.New()
	hasher.Write([]byte(code))
	codeHash := hex.EncodeToString(hasher.Sum(nil))

	if codeHash != verification.CodeHash {
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Geçersiz doğrulama kodu.")
		return
	}

	// Mark as verified
	now := time.Now()
	verification.UsedAt = &now
	if err := h.db.WithContext(c.Request.Context()).Save(&verification).Error; err != nil {
		h.logger.Error("failed to mark verification as used", "error", err, "verification_id", verification.ID)
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama sırasında bir hata oluştu.")
		return
	}

	if err := h.db.WithContext(c.Request.Context()).Model(&currentUser).Update("phone_verified_at", &now).Error; err != nil {
		h.logger.Error("failed to update user phone_verified_at", "error", err, "user_id", currentUser.ID)
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama sırasında bir hata oluştu.")
		return
	}

	render.RedirectWithFlash(c, h.flash, "/account", view.FlashSuccess, "Telefon numaranız başarıyla doğrulandı.")
}

func (h *SmsHandler) PostSendCode(c *gin.Context) {
	// 1. Get user from context
	ctxUser, ok := middleware.CurrentUser(c)
	if !ok {
		render.Redirect(c, "/login")
		return
	}
	var currentUser auth.User
	if err := h.db.First(&currentUser, "id = ?", ctxUser.ID).Error; err != nil {
		render.Redirect(c, "/login")
		return
	}

	// 2. Check for phone number
	if currentUser.PhoneE164 == nil || *currentUser.PhoneE164 == "" {
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Lütfen önce telefon numarasını ekleyin.")
		return
	}

	// 3. Use verification service if available
	if h.verificationSvc != nil {
		_, err := h.verificationSvc.StartPhoneVerification(c.Request.Context(), currentUser.ID, *currentUser.PhoneE164)
		if err != nil {
			h.logger.Error("failed to start phone verification", "error", err, "user_id", currentUser.ID)
			render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama kodu gönderilemedi: "+err.Error())
			return
		}
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashSuccess, "Doğrulama kodu SMS ile gönderildi. Lütfen kontrol edin.")
		return
	}

	// Fallback: Generate OTP without service
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	hasher := sha256.New()
	hasher.Write([]byte(otp))
	otpHash := hex.EncodeToString(hasher.Sum(nil))

	// 4. Store verification record
	verification := sms.PhoneVerification{
		UserID:    currentUser.ID,
		PhoneE164: *currentUser.PhoneE164,
		CodeHash:  otpHash,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&verification).Error; err != nil {
		h.logger.Error("failed to create phone verification record", "error", err, "user_id", currentUser.ID)
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama kodu gönderilemedi.")
		return
	}

	// 5. Enqueue SMS
	_, err := h.smsRepo.Enqueue(c.Request.Context(), sms.EnqueueOptions{
		ToPhoneE164: *currentUser.PhoneE164,
		Template:    "otp",
		Payload: map[string]interface{}{
			"code": otp,
		},
	})
	if err != nil {
		h.logger.Error("failed to enqueue OTP sms", "error", err, "user_id", currentUser.ID)
		render.RedirectWithFlash(c, h.flash, "/account", view.FlashError, "Doğrulama kodu gönderilemedi.")
		return
	}

	render.RedirectWithFlash(c, h.flash, "/account", view.FlashSuccess, "A verification code has been sent to your phone.")
}
