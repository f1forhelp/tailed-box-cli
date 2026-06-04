package meshcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	ProtocolSuite        = "TBX-MESH-V1"
	CipherSuiteAES256GCM = "aes-256-gcm"
	HKDFInfo             = "tailedbox mesh v1 session"
	ProtocolVersion      = 1

	SessionKeySize = 32
	NoncePrefixLen = 4
	NonceLen       = 12
)

type Transcript struct {
	Version                      int       `json:"version"`
	Suite                        string    `json:"suite"`
	ClusterID                    string    `json:"cluster_id"`
	InitiatorNodeID              string    `json:"initiator_node_id"`
	InitiatorRole                string    `json:"initiator_role"`
	InitiatorIdentityFingerprint string    `json:"initiator_identity_fingerprint"`
	InitiatorEphemeralPublic     []byte    `json:"initiator_ephemeral_public"`
	InitiatorNonce               []byte    `json:"initiator_nonce"`
	InitiatorTime                time.Time `json:"initiator_time"`
	ResponderNodeID              string    `json:"responder_node_id"`
	ResponderRole                string    `json:"responder_role"`
	ResponderIdentityFingerprint string    `json:"responder_identity_fingerprint"`
	ResponderEphemeralPublic     []byte    `json:"responder_ephemeral_public"`
	ResponderNonce               []byte    `json:"responder_nonce"`
	ResponderTime                time.Time `json:"responder_time"`
	CipherSuite                  string    `json:"cipher_suite"`
}

type SessionKeys struct {
	InitiatorToResponderKey []byte
	ResponderToInitiatorKey []byte
	InitiatorNoncePrefix    []byte
	ResponderNoncePrefix    []byte
	TranscriptHash          []byte
}

func GenerateEphemeral() (*ecdh.PrivateKey, []byte, error) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate x25519 ephemeral key: %w", err)
	}
	return privateKey, privateKey.PublicKey().Bytes(), nil
}

func CanonicalTranscript(transcript Transcript) ([]byte, error) {
	normalized := normalizeTranscript(transcript)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal mesh transcript: %w", err)
	}
	return data, nil
}

func TranscriptHash(transcript Transcript) ([]byte, error) {
	data, err := CanonicalTranscript(transcript)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	return sum[:], nil
}

func SignTranscript(privateKey ed25519.PrivateKey, transcript Transcript) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("ed25519 private key is required for transcript signing")
	}
	data, err := CanonicalTranscript(transcript)
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(privateKey, data), nil
}

func VerifyTranscript(publicKey ed25519.PublicKey, transcript Transcript, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return false
	}
	data, err := CanonicalTranscript(transcript)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, data, signature)
}

func DeriveSessionKeys(privateKey *ecdh.PrivateKey, peerPublicKey []byte, transcript Transcript) (SessionKeys, error) {
	if privateKey == nil {
		return SessionKeys{}, errors.New("x25519 private key is required")
	}
	peer, err := ecdh.X25519().NewPublicKey(peerPublicKey)
	if err != nil {
		return SessionKeys{}, fmt.Errorf("parse x25519 peer public key: %w", err)
	}
	sharedSecret, err := privateKey.ECDH(peer)
	if err != nil {
		return SessionKeys{}, fmt.Errorf("derive x25519 shared secret: %w", err)
	}
	transcriptHash, err := TranscriptHash(transcript)
	if err != nil {
		return SessionKeys{}, err
	}
	keyMaterial, err := hkdf.Key(sha256.New, sharedSecret, transcriptHash, HKDFInfo, SessionKeySize*2+NoncePrefixLen*2)
	if err != nil {
		return SessionKeys{}, fmt.Errorf("derive mesh session keys: %w", err)
	}

	offset := 0
	initiatorKey := cloneBytes(keyMaterial[offset : offset+SessionKeySize])
	offset += SessionKeySize
	responderKey := cloneBytes(keyMaterial[offset : offset+SessionKeySize])
	offset += SessionKeySize
	initiatorNoncePrefix := cloneBytes(keyMaterial[offset : offset+NoncePrefixLen])
	offset += NoncePrefixLen
	responderNoncePrefix := cloneBytes(keyMaterial[offset : offset+NoncePrefixLen])

	return SessionKeys{
		InitiatorToResponderKey: initiatorKey,
		ResponderToInitiatorKey: responderKey,
		InitiatorNoncePrefix:    initiatorNoncePrefix,
		ResponderNoncePrefix:    responderNoncePrefix,
		TranscriptHash:          cloneBytes(transcriptHash),
	}, nil
}

func NewAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != SessionKeySize {
		return nil, fmt.Errorf("mesh session key must be %d bytes", SessionKeySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create aes-gcm: %w", err)
	}
	return aead, nil
}

func Nonce(prefix []byte, sequence uint64) ([]byte, error) {
	if len(prefix) != NoncePrefixLen {
		return nil, fmt.Errorf("mesh nonce prefix must be %d bytes", NoncePrefixLen)
	}
	nonce := make([]byte, NonceLen)
	copy(nonce[0:NoncePrefixLen], prefix)
	binary.BigEndian.PutUint64(nonce[NoncePrefixLen:], sequence)
	return nonce, nil
}

func normalizeTranscript(transcript Transcript) Transcript {
	if transcript.Version == 0 {
		transcript.Version = ProtocolVersion
	}
	if transcript.Suite == "" {
		transcript.Suite = ProtocolSuite
	}
	if transcript.CipherSuite == "" {
		transcript.CipherSuite = CipherSuiteAES256GCM
	}
	return transcript
}

func cloneBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}
