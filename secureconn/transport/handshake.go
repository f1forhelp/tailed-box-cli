package transport

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	meshcrypto "github.com/tailedbox/secureconn/crypto"
	"github.com/tailedbox/secureconn/identity"
	"github.com/tailedbox/secureconn/protocol"
)

const (
	handshakeVersion = 1
	modeEnrolled     = "enrolled"
)

type clientHello struct {
	Version              int                     `json:"version"`
	Mode                 string                  `json:"mode"`
	ClusterID            string                  `json:"cluster_id"`
	NodeID               string                  `json:"node_id"`
	Role                 string                  `json:"role"`
	IdentityFingerprint  string                  `json:"identity_fingerprint"`
	PublicIdentity       identity.PublicIdentity `json:"public_identity"`
	EphemeralPublic      []byte                  `json:"ephemeral_public"`
	Nonce                []byte                  `json:"nonce"`
	SentAt               time.Time               `json:"sent_at"`
	SupportedCipherSuite []string                `json:"supported_cipher_suites"`
}

type serverHello struct {
	Version             int                     `json:"version"`
	Mode                string                  `json:"mode"`
	ClusterID           string                  `json:"cluster_id"`
	NodeID              string                  `json:"node_id"`
	Role                string                  `json:"role"`
	IdentityFingerprint string                  `json:"identity_fingerprint"`
	PublicIdentity      identity.PublicIdentity `json:"public_identity"`
	EphemeralPublic     []byte                  `json:"ephemeral_public"`
	Nonce               []byte                  `json:"nonce"`
	SentAt              time.Time               `json:"sent_at"`
	CipherSuite         string                  `json:"cipher_suite"`
	SessionID           protocol.SessionID      `json:"session_id"`
	Signature           []byte                  `json:"signature"`
}

type clientAuth struct {
	Version             int       `json:"version"`
	ClusterID           string    `json:"cluster_id"`
	NodeID              string    `json:"node_id"`
	Role                string    `json:"role"`
	IdentityFingerprint string    `json:"identity_fingerprint"`
	SentAt              time.Time `json:"sent_at"`
	Signature           []byte    `json:"signature"`
}

func (h clientHello) validate(now time.Time) error {
	if h.Version != handshakeVersion {
		return fmt.Errorf("unsupported client hello version %d", h.Version)
	}
	if h.Mode != modeEnrolled {
		return fmt.Errorf("unsupported mesh handshake mode %q", h.Mode)
	}
	if h.ClusterID == "" || h.NodeID == "" || h.Role == "" || h.IdentityFingerprint == "" {
		return errors.New("client hello is missing required identity fields")
	}
	if len(h.EphemeralPublic) == 0 || len(h.Nonce) == 0 {
		return errors.New("client hello is missing ephemeral key material")
	}
	if err := validateTimestamp(now, h.SentAt); err != nil {
		return err
	}
	_, err := publicKeyFromIdentity(h.PublicIdentity, h.NodeID, h.IdentityFingerprint)
	return err
}

func (h serverHello) validate(now time.Time) error {
	if h.Version != handshakeVersion {
		return fmt.Errorf("unsupported server hello version %d", h.Version)
	}
	if h.Mode != modeEnrolled {
		return fmt.Errorf("unsupported mesh handshake mode %q", h.Mode)
	}
	if h.ClusterID == "" || h.NodeID == "" || h.Role == "" || h.IdentityFingerprint == "" {
		return errors.New("server hello is missing required identity fields")
	}
	if len(h.EphemeralPublic) == 0 || len(h.Nonce) == 0 || len(h.Signature) != ed25519.SignatureSize {
		return errors.New("server hello is missing signed ephemeral key material")
	}
	if h.CipherSuite != meshcrypto.CipherSuiteAES256GCM {
		return fmt.Errorf("unsupported mesh cipher suite %q", h.CipherSuite)
	}
	if err := validateTimestamp(now, h.SentAt); err != nil {
		return err
	}
	_, err := publicKeyFromIdentity(h.PublicIdentity, h.NodeID, h.IdentityFingerprint)
	return err
}

func (a clientAuth) validate(now time.Time) error {
	if a.Version != handshakeVersion {
		return fmt.Errorf("unsupported client auth version %d", a.Version)
	}
	if a.ClusterID == "" || a.NodeID == "" || a.Role == "" || a.IdentityFingerprint == "" {
		return errors.New("client auth is missing required identity fields")
	}
	if len(a.Signature) != ed25519.SignatureSize {
		return errors.New("client auth signature is required")
	}
	return validateTimestamp(now, a.SentAt)
}

func buildTranscript(hello clientHello, response serverHello) meshcrypto.Transcript {
	return meshcrypto.Transcript{
		Version:                      meshcrypto.ProtocolVersion,
		Suite:                        meshcrypto.ProtocolSuite,
		ClusterID:                    hello.ClusterID,
		InitiatorNodeID:              hello.NodeID,
		InitiatorRole:                hello.Role,
		InitiatorIdentityFingerprint: hello.IdentityFingerprint,
		InitiatorEphemeralPublic:     hello.EphemeralPublic,
		InitiatorNonce:               hello.Nonce,
		InitiatorTime:                hello.SentAt.UTC(),
		ResponderNodeID:              response.NodeID,
		ResponderRole:                response.Role,
		ResponderIdentityFingerprint: response.IdentityFingerprint,
		ResponderEphemeralPublic:     response.EphemeralPublic,
		ResponderNonce:               response.Nonce,
		ResponderTime:                response.SentAt.UTC(),
		CipherSuite:                  response.CipherSuite,
	}
}

func publicKeyFromIdentity(publicIdentity identity.PublicIdentity, nodeID, fingerprint string) (ed25519.PublicKey, error) {
	return identity.PublicKeyFromIdentity(publicIdentity, nodeID, fingerprint)
}
