package middleware

import (
	"crypto/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionCfg holds configuration for session middleware.
type SessionCfg struct {
	DB         *gorm.DB
	CookieName string
	Secure     bool
	TTL        time.Duration
}

// Session is a database-backed session model.
type Session struct {
	ID         string    `gorm:"primaryKey;type:char(36)"`
	UserID     string    `gorm:"type:char(36);not null;index:ix_sessions_user_id"`
	TokenHash  []byte    `gorm:"type:binary(32);not null;uniqueIndex:ux_sessions_token_hash"`
	ExpiresAt  time.Time `gorm:"type:datetime(3);not null"`
	CreatedAt  time.Time `gorm:"type:datetime(3);not null"`
	UpdatedAt  time.Time `gorm:"type:datetime(3);not null"`
	LastSeenAt time.Time `gorm:"type:datetime(3);not null"`
}

func (Session) TableName() string { return "sessions" }

// SessionMiddleware loads/creates a session from the database and sets user info in context.
func SessionMiddleware(cfg SessionCfg) gin.HandlerFunc {
	// Don't use AutoMigrate; migrations are handled by goose migration files
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(cfg.CookieName)
		if err != nil || sessionID == "" {
			c.Next()
			return
		}

		// Load session from DB
		var sess Session
		if err := cfg.DB.Where("id = ? AND expires_at > ?", sessionID, time.Now()).First(&sess).Error; err != nil {
			// Invalid or expired session, clear cookie
			c.SetCookie(cfg.CookieName, "", -1, "/", "", cfg.Secure, true)
			c.Next()
			return
		}

		// Store session in context
		c.Set("session", &sess)
		c.Set("user_id", sess.UserID)

		// Load user email + role from DB for context
		var userEmail string
		var userRole string
		row := cfg.DB.Table("users").Select("email", "role").Where("id = ?", sess.UserID).Row()
		if err := row.Scan(&userEmail, &userRole); err == nil {
			c.Set("user_email", userEmail)
			c.Set("user_role", userRole)
		}

		c.Next()
	}
}

// CreateSession creates a new session for the given user.
func CreateSession(cfg SessionCfg, userID string) (*Session, error) {
	tokenHash, _ := generateTokenHash()
	sess := &Session{
		ID:         generateSessionID(),
		UserID:     userID,
		TokenHash:  tokenHash,
		ExpiresAt:  time.Now().Add(cfg.TTL),
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
	}
	if err := cfg.DB.Create(sess).Error; err != nil {
		return nil, err
	}
	return sess, nil
}

// DeleteSession removes a session by ID.
func DeleteSession(cfg SessionCfg, sessionID string) error {
	return cfg.DB.Delete(&Session{}, "id = ?", sessionID).Error
}

func generateSessionID() string {
	return uuid.New().String()
}

func generateTokenHash() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return b, err
}

// ContextUser represents the authenticated user stored in request context.
type ContextUser struct {
	ID    string
	Email string
	Role  string
}

// CurrentUser retrieves the authenticated user from the gin context.
// Returns the user and true if authenticated, or zero value and false otherwise.
func CurrentUser(c *gin.Context) (ContextUser, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return ContextUser{}, false
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return ContextUser{}, false
	}

	var emailStr, roleStr string
	if email, ok := c.Get("user_email"); ok && email != nil {
		emailStr, _ = email.(string)
	}
	if role, ok := c.Get("user_role"); ok && role != nil {
		roleStr, _ = role.(string)
	}

	return ContextUser{
		ID:    userID,
		Email: emailStr,
		Role:  roleStr,
	}, true
}
