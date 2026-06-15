package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
)

func SHA256(data []byte) []byte {
	sum := sha256.Sum256(data)
	out := make([]byte, len(sum))
	copy(out, sum[:])
	return out
}

func HMACSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func ConstantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
