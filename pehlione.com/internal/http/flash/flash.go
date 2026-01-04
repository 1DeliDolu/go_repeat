package flash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"pehlione.com/app/pkg/view"
)

var ErrInvalid = errors.New("invalid flash cookie")

type Codec struct {
	Secret     []byte
	CookieName string
	Secure     bool
}

func NewCodec(secret []byte, cookieName string, secure bool) *Codec {
	return &Codec{Secret: secret, CookieName: cookieName, Secure: secure}
}

// value format: base64(json).base64(hmac)
func (c *Codec) Encode(f view.Flash) (string, error) {
	b, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	sig := sign(c.Secret, payload)
	return payload + "." + sig, nil
}

func (c *Codec) Decode(v string) (*view.Flash, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 2 {
		return nil, ErrInvalid
	}
	payload, sig := parts[0], parts[1]
	if !verify(c.Secret, payload, sig) {
		return nil, ErrInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, ErrInvalid
	}
	var f view.Flash
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(f.Message) == "" {
		return nil, ErrInvalid
	}
	return &f, nil
}

func (c *Codec) CookieMaxAge() int {
	// Flash kısa ömürlü: 2 dakika yeterli (redirect sonrası okunsun)
	return int((2 * time.Minute).Seconds())
}

func sign(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}

func verify(secret []byte, payload, sig string) bool {
	expected := sign(secret, payload)
	return hmac.Equal([]byte(expected), []byte(sig))
}
