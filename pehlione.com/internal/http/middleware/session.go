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
		var firstName *string
		var lastName *string
		var phoneE164 *string
		var address *string
		var emailVerifiedAt *time.Time
		row := cfg.DB.Table("users").Select("email", "role", "first_name", "last_name", "phone_e164", "address", "email_verified_at").Where("id = ?", sess.UserID).Row()
		if err := row.Scan(&userEmail, &userRole, &firstName, &lastName, &phoneE164, &address, &emailVerifiedAt); err == nil {
			c.Set("user_email", userEmail)
			c.Set("user_role", userRole)
			c.Set("user_first_name", firstName)
			c.Set("user_last_name", lastName)
			c.Set("user_phone", phoneE164)
			c.Set("user_address", address)
			c.Set("user_email_verified_at", emailVerifiedAt)
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
	ID              string
	Email           string
	Role            string
	FirstName       *string
	LastName        *string
	PhoneE164       *string
	Address         *string
	EmailVerifiedAt *time.Time
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
	var firstName, lastName, phoneE164, address *string
	var emailVerifiedAt *time.Time
	if email, ok := c.Get("user_email"); ok && email != nil {
		emailStr, _ = email.(string)
	}
	if role, ok := c.Get("user_role"); ok && role != nil {
		roleStr, _ = role.(string)
	}
	if fn, ok := c.Get("user_first_name"); ok && fn != nil {
		firstName, _ = fn.(*string)
	}
	if ln, ok := c.Get("user_last_name"); ok && ln != nil {
		lastName, _ = ln.(*string)
	}
	if phone, ok := c.Get("user_phone"); ok && phone != nil {
		phoneE164, _ = phone.(*string)
	}
	if addr, ok := c.Get("user_address"); ok && addr != nil {
		address, _ = addr.(*string)
	}
	if eva, ok := c.Get("user_email_verified_at"); ok && eva != nil {
		emailVerifiedAt, _ = eva.(*time.Time)
	}

	return ContextUser{
		ID:              userID,
		Email:           emailStr,
		Role:            roleStr,
		FirstName:       firstName,
		LastName:        lastName,
		PhoneE164:       phoneE164,
		Address:         address,
		EmailVerifiedAt: emailVerifiedAt,
	}, true
}
