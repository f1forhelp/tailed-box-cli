package enrollment

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
)

const (
	JoinCodePrefix = "tbxjc1"
	HashSHA256     = "sha256"
)

type JoinCodePayload struct {
	Version           int       `json:"version"`
	CodeID            string    `json:"code_id"`
	AllowedRole       string    `json:"allowed_role"`
	ClusterID         string    `json:"cluster_id"`
	ClusterName       string    `json:"cluster_name"`
	IssuerNodeID      string    `json:"issuer_node_id"`
	IssuerFingerprint string    `json:"issuer_fingerprint"`
	IssuedAt          time.Time `json:"issued_at"`
	ExpiresAt         time.Time `json:"expires_at"`
}

type ParsedJoinCode struct {
	Raw     string
	Payload JoinCodePayload
	Secret  string
}

func NewJoinCode(payload JoinCodePayload) (string, string, error) {
	if err := validatePayload(payload); err != nil {
		return "", "", err
	}
	secret, err := newSecret()
	if err != nil {
		return "", "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal join-code payload: %w", err)
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return JoinCodePrefix + "." + payloadPart + "." + secret, secretHash(payload.CodeID, secret), nil
}

func ParseJoinCode(raw string) (ParsedJoinCode, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedJoinCode{}, errors.New("join code is required")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || parts[0] != JoinCodePrefix {
		return ParsedJoinCode{}, errors.New("invalid join code format")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ParsedJoinCode{}, errors.New("invalid join code payload")
	}
	var payload JoinCodePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return ParsedJoinCode{}, errors.New("invalid join code payload")
	}
	if err := validatePayload(payload); err != nil {
		return ParsedJoinCode{}, err
	}
	if len(parts[2]) < 32 {
		return ParsedJoinCode{}, errors.New("invalid join code secret")
	}
	return ParsedJoinCode{Raw: raw, Payload: payload, Secret: parts[2]}, nil
}

func HashForParsed(parsed ParsedJoinCode) string {
	return secretHash(parsed.Payload.CodeID, parsed.Secret)
}

func NewCodeID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate join code id: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return "jc_" + strings.ToLower(encoded), nil
}

func validatePayload(payload JoinCodePayload) error {
	if payload.Version != 1 {
		return fmt.Errorf("unsupported join code version %d", payload.Version)
	}
	if payload.CodeID == "" {
		return errors.New("join code id is required")
	}
	if !config.ValidRole(payload.AllowedRole) {
		return fmt.Errorf("unsupported join role %q", payload.AllowedRole)
	}
	if payload.ClusterID == "" {
		return errors.New("join code cluster id is required")
	}
	if payload.IssuerNodeID == "" {
		return errors.New("join code issuer node id is required")
	}
	if payload.IssuerFingerprint == "" {
		return errors.New("join code issuer fingerprint is required")
	}
	if payload.IssuedAt.IsZero() || payload.ExpiresAt.IsZero() || !payload.ExpiresAt.After(payload.IssuedAt) {
		return errors.New("join code expiry is invalid")
	}
	return nil
}

func newSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate join code secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func secretHash(codeID, secret string) string {
	sum := sha256.Sum256([]byte(codeID + "." + secret))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}
