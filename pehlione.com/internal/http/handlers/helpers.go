package handlers

import (
	"crypto/rand"
	"encoding/hex"
)

func randHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
