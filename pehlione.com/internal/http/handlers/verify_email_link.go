package handlers

import (
	"crypto/sha256"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VerifyEmailHandler struct {
	db *gorm.DB
}

func NewVerifyEmailHandler(db *gorm.DB) *VerifyEmailHandler {
	return &VerifyEmailHandler{db: db}
}

func (h *VerifyEmailHandler) Get(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "missing token")
		return
	}

	sum := sha256.Sum256([]byte(token))

	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var ev struct {
			ID        string
			UserID    string
			ExpiresAt time.Time
			UsedAt    *time.Time
		}
		if err := tx.Raw(`
			SELECT id, user_id, expires_at, used_at
			FROM email_verifications
			WHERE code_hash = ?
			LIMIT 1
			FOR UPDATE
		`, sum[:]).Scan(&ev).Error; err != nil {
			return err
		}
		if ev.ID == "" || ev.UsedAt != nil || time.Now().After(ev.ExpiresAt) {
			return gorm.ErrRecordNotFound
		}

		now := time.Now()
		if err := tx.Exec(`UPDATE email_verifications SET used_at=? WHERE id=?`, now, ev.ID).Error; err != nil {
			return err
		}

		return tx.Table("users").
			Where("id = ?", ev.UserID).
			Updates(map[string]any{
				"email_verified_at": now,
				"status":            "active",
			}).Error
	})

	if err != nil {
		c.String(http.StatusBadRequest, "invalid or expired token")
		return
	}

	c.Redirect(http.StatusSeeOther, "/login?verified=1")
}
