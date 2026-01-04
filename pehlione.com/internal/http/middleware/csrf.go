package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSRFCfg holds configuration for CSRF protection.
type CSRFCfg struct {
	CookieName string
	Secure     bool
}

// CSRF implements double-submit cookie pattern.
func CSRF(cfg CSRFCfg) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip webhook endpoints (signature verification is security layer)
		if strings.HasPrefix(c.Request.URL.Path, "/webhooks/") {
			c.Next()
			return
		}

		// Skip API endpoints (HTMX can include CSRF token via header if needed)
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}

		// Read or generate CSRF token
		token, err := c.Cookie(cfg.CookieName)
		if err != nil || token == "" {
			token = generateCSRFToken()
			c.SetCookie(cfg.CookieName, token, 0, "/", "", cfg.Secure, false) // httpOnly=false so JS can read
		}

		// Store token in context for templates
		c.Set("csrf_token", token)

		// For safe methods, just continue
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Validate token for unsafe methods
		formToken := c.PostForm("csrf_token")
		if formToken == "" {
			formToken = c.GetHeader("X-CSRF-Token")
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(formToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid CSRF token"})
			return
		}

		c.Next()
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// GetCSRFToken retrieves the CSRF token from context.
func GetCSRFToken(c *gin.Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		if s, ok := token.(string); ok {
			return s
		}
	}
	return ""
}
