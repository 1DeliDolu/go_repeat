package cartcookie

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var ErrInvalid = errors.New("invalid cart cookie")

type Codec struct {
	Secret     []byte
	CookieName string
	Secure     bool
}

func New(secret []byte, name string, secure bool) *Codec {
	return &Codec{Secret: secret, CookieName: name, Secure: secure}
}

// value format: cartID.base64(hmac(cartID))
func (c *Codec) Encode(cartID string) string {
	sig := sign(c.Secret, cartID)
	return cartID + "." + sig
}

func (c *Codec) Decode(v string) (string, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 2 {
		return "", ErrInvalid
	}
	id := parts[0]
	if id == "" {
		return "", ErrInvalid
	}
	if !verify(c.Secret, id, parts[1]) {
		return "", ErrInvalid
	}
	return id, nil
}

func (c *Codec) GetCartID(ctx *gin.Context) (string, bool) {
	v, err := ctx.Cookie(c.CookieName)
	if err != nil || v == "" {
		return "", false
	}
	id, err := c.Decode(v)
	if err != nil {
		c.Clear(ctx)
		return "", false
	}
	return id, true
}

func (c *Codec) Set(ctx *gin.Context, cartID string) {
	val := c.Encode(cartID)
	maxAge := int((30 * 24 * time.Hour).Seconds())
	ctx.SetSameSite(2) // Lax
	ctx.SetCookie(c.CookieName, val, maxAge, "/", "", c.Secure, true)
}

func (c *Codec) Clear(ctx *gin.Context) {
	ctx.SetSameSite(2) // Lax
	ctx.SetCookie(c.CookieName, "", -1, "/", "", c.Secure, true)
}

func sign(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

func verify(secret []byte, payload, sig string) bool {
	return hmac.Equal([]byte(sign(secret, payload)), []byte(sig))
}
