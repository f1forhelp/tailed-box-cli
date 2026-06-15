package crypto

import (
	"encoding/base32"
	"encoding/base64"
	"strings"
)

var base32NoPadding = base32.StdEncoding.WithPadding(base32.NoPadding)

func Base32NoPaddingEncode(data []byte) string {
	return base32NoPadding.EncodeToString(data)
}

func Base32NoPaddingDecode(value string) ([]byte, error) {
	return base32NoPadding.DecodeString(NormalizeBase32(value))
}

func NormalizeBase32(value string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(value) {
		switch r {
		case '-', ' ', '\t', '\n', '\r':
			continue
		default:
			builder.WriteRune(r)
		}
	}
	return strings.ToUpper(builder.String())
}

func Base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func Base64URLDecode(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
}
