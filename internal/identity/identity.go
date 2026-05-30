package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	AlgorithmEd25519 = "ed25519"
	privateKeyType   = "TAILEDBOX ED25519 PRIVATE KEY"
)

type PublicIdentity struct {
	Version              int       `json:"version"`
	NodeID               string    `json:"node_id"`
	Algorithm            string    `json:"algorithm"`
	PublicKey            string    `json:"public_key"`
	PublicKeyFingerprint string    `json:"public_key_fingerprint"`
	CreatedAt            time.Time `json:"created_at"`
}

type EnsureResult struct {
	Created               bool
	PublicIdentityChanged bool
	PublicIdentity        PublicIdentity
	PrivateKeyFile        string
	PublicIdentityFile    string
}

func Ensure(cfg *config.Config, now time.Time) (EnsureResult, error) {
	if cfg == nil {
		return EnsureResult{}, errors.New("config is nil")
	}
	if cfg.Node.ID == "" {
		return EnsureResult{}, errors.New("node id is required before identity initialization")
	}
	if cfg.Paths.IdentityPrivateKeyFile == "" || cfg.Paths.IdentityPublicKeyFile == "" {
		return EnsureResult{}, errors.New("identity paths are not configured")
	}
	if err := secrets.EnsurePrivateDir(cfg.Paths.SecretsDir); err != nil {
		return EnsureResult{}, err
	}

	privateKey, createdAt, created, err := loadOrCreatePrivateKey(cfg.Paths.IdentityPrivateKeyFile, now)
	if err != nil {
		return EnsureResult{}, err
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return EnsureResult{}, errors.New("identity private key does not expose an ed25519 public key")
	}
	publicIdentity := PublicIdentity{
		Version:              1,
		NodeID:               cfg.Node.ID,
		Algorithm:            AlgorithmEd25519,
		PublicKey:            base64.RawStdEncoding.EncodeToString(publicKey),
		PublicKeyFingerprint: Fingerprint(publicKey),
		CreatedAt:            createdAt.UTC(),
	}

	if err := validateExistingPublicIdentity(cfg.Paths.IdentityPublicKeyFile, publicIdentity); err != nil {
		return EnsureResult{}, err
	}
	publicChanged, err := secrets.WriteJSONAtomic(cfg.Paths.IdentityPublicKeyFile, publicIdentity)
	if err != nil {
		return EnsureResult{}, err
	}

	cfg.Node.Identity = config.IdentityConfig{
		Algorithm:            publicIdentity.Algorithm,
		PublicKeyFingerprint: publicIdentity.PublicKeyFingerprint,
		PublicKeyFile:        cfg.Paths.IdentityPublicKeyFile,
		CreatedAt:            publicIdentity.CreatedAt,
	}

	return EnsureResult{
		Created:               created,
		PublicIdentityChanged: publicChanged,
		PublicIdentity:        publicIdentity,
		PrivateKeyFile:        cfg.Paths.IdentityPrivateKeyFile,
		PublicIdentityFile:    cfg.Paths.IdentityPublicKeyFile,
	}, nil
}

func Fingerprint(publicKey ed25519.PublicKey) string {
	sum := sha256.Sum256(publicKey)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:16])
	return "tbx1_" + strings.ToLower(encoded)
}

func loadOrCreatePrivateKey(path string, now time.Time) (ed25519.PrivateKey, time.Time, bool, error) {
	privateKey, createdAt, err := LoadPrivateKey(path)
	if err == nil {
		return privateKey, createdAt, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, time.Time{}, false, err
	}

	_, privateKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("generate node identity key: %w", err)
	}
	data, err := marshalPrivateKey(privateKey)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	created, err := secrets.WriteFileNew(path, data)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	if !created {
		privateKey, createdAt, err := LoadPrivateKey(path)
		return privateKey, createdAt, false, err
	}
	return privateKey, now.UTC(), true, nil
}

func LoadPrivateKey(path string) (ed25519.PrivateKey, time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, time.Time{}, fmt.Errorf("parse identity private key %q: missing pem block", path)
	}
	if block.Type != privateKeyType {
		return nil, time.Time{}, fmt.Errorf("parse identity private key %q: unexpected pem type %q", path, block.Type)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("parse identity private key %q: %w", path, err)
	}
	privateKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, time.Time{}, fmt.Errorf("parse identity private key %q: expected ed25519 key", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat identity private key %q: %w", path, err)
	}
	if err := os.Chmod(path, secrets.PrivateFileMode); err != nil {
		return nil, time.Time{}, fmt.Errorf("secure identity private key %q: %w", path, err)
	}
	return privateKey, info.ModTime().UTC(), nil
}

func marshalPrivateKey(privateKey ed25519.PrivateKey) ([]byte, error) {
	data, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal identity private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: privateKeyType, Bytes: data}), nil
}

func validateExistingPublicIdentity(path string, expected PublicIdentity) error {
	var existing PublicIdentity
	if err := secrets.ReadJSON(path, &existing); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if existing.NodeID != expected.NodeID {
		return fmt.Errorf("identity public metadata belongs to node %s, expected %s", existing.NodeID, expected.NodeID)
	}
	if existing.Algorithm != expected.Algorithm {
		return fmt.Errorf("identity public metadata uses algorithm %s, expected %s", existing.Algorithm, expected.Algorithm)
	}
	if existing.PublicKeyFingerprint != expected.PublicKeyFingerprint {
		return fmt.Errorf("identity public metadata fingerprint mismatch")
	}
	return nil
}
