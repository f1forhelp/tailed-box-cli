package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const AlgorithmEd25519 = "ed25519"

type PublicIdentity struct {
	Version              int       `json:"version"`
	NodeID               string    `json:"node_id"`
	Algorithm            string    `json:"algorithm"`
	PublicKey            string    `json:"public_key"`
	PublicKeyFingerprint string    `json:"public_key_fingerprint"`
	CreatedAt            time.Time `json:"created_at"`
}

func Fingerprint(publicKey ed25519.PublicKey) string {
	sum := sha256.Sum256(publicKey)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:16])
	return "tbx1_" + strings.ToLower(encoded)
}

func PublicKeyFromIdentity(publicIdentity PublicIdentity, nodeID, fingerprint string) (ed25519.PublicKey, error) {
	if publicIdentity.NodeID != nodeID {
		return nil, fmt.Errorf("public identity belongs to node %s, expected %s", publicIdentity.NodeID, nodeID)
	}
	if publicIdentity.Algorithm != AlgorithmEd25519 {
		return nil, fmt.Errorf("public identity uses unsupported algorithm %q", publicIdentity.Algorithm)
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(publicIdentity.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode public identity key: %w", err)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public identity key has %d bytes, expected %d", len(publicKey), ed25519.PublicKeySize)
	}
	if got := Fingerprint(ed25519.PublicKey(publicKey)); got != fingerprint || got != publicIdentity.PublicKeyFingerprint {
		return nil, fmt.Errorf("public identity fingerprint mismatch")
	}
	return ed25519.PublicKey(publicKey), nil
}
