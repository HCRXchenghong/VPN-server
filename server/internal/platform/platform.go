package platform

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func NewID(prefix string) string {
	return prefix + "_" + randomHex(8)
}

func NewToken(prefix string) string {
	return prefix + "_" + randomHex(24)
}

func HashPassword(password string) string {
	sum := sha256.Sum256([]byte("vpn-platform:" + password))
	return hex.EncodeToString(sum[:])
}

func PasswordMatches(password, hashed string) bool {
	return strings.EqualFold(HashPassword(password), hashed)
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("random read failed: %w", err))
	}
	return hex.EncodeToString(buf)
}
