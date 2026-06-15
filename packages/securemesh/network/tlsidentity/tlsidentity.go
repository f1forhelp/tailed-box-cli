package tlsidentity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

const (
	ALPN            = "tailed-box-mesh/1"
	metadataVersion = 1
	certificateTTL  = 30 * 24 * time.Hour
)

var (
	ErrInvalidOptions     = errors.New("invalid tls identity options")
	ErrMissingCertificate = errors.New("missing peer certificate")
	ErrInvalidCertificate = errors.New("invalid peer certificate")
	ErrMissingMetadata    = errors.New("missing mesh identity metadata")
	ErrUnknownPeer        = errors.New("unknown peer")
	ErrWrongNetwork       = errors.New("peer is for a different network")
	ErrWrongRole          = errors.New("peer has wrong role")
	ErrRevokedPeer        = errors.New("peer is revoked")
	ErrPublicKeyMismatch  = errors.New("peer public key mismatch")
)

var metadataExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 55555, 1, 1}

type CertificateMetadata struct {
	Version    int                   `json:"version"`
	NodeID     identity.NodeID       `json:"node_id"`
	NetworkID  identity.NetworkID    `json:"network_id"`
	Role       identity.Role         `json:"role"`
	PublicKeys identity.PublicKeySet `json:"public_keys"`
	CreatedAt  time.Time             `json:"created_at"`
}

func (m CertificateMetadata) Validate() error {
	if m.Version != metadataVersion {
		return fmt.Errorf("%w: unsupported metadata version %d", ErrInvalidCertificate, m.Version)
	}
	if err := m.NodeID.Validate(); err != nil {
		return err
	}
	if err := m.NetworkID.Validate(); err != nil {
		return err
	}
	if err := m.Role.Validate(); err != nil {
		return err
	}
	if err := m.PublicKeys.Validate(); err != nil {
		return err
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("%w: metadata created_at is required", ErrInvalidCertificate)
	}
	derived, err := identity.DeriveNodeID(m.PublicKeys)
	if err != nil {
		return err
	}
	if derived != m.NodeID {
		return fmt.Errorf("%w: metadata node id does not match public keys", ErrInvalidCertificate)
	}
	return nil
}

func NewCertificate(local identity.Identity, now time.Time) (tls.Certificate, error) {
	if err := local.Validate(); err != nil {
		return tls.Certificate{}, err
	}
	privateKey, err := signingPrivateKey(local)
	if err != nil {
		return tls.Certificate{}, err
	}
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return tls.Certificate{}, fmt.Errorf("%w: invalid ed25519 public key", ErrInvalidCertificate)
	}
	if !bytes.Equal(publicKey, local.PublicKeys.Signing.Bytes) {
		return tls.Certificate{}, fmt.Errorf("%w: signing public key does not match private key", ErrInvalidCertificate)
	}

	now = normalizeTime(now)
	metadata := CertificateMetadata{
		Version:    metadataVersion,
		NodeID:     local.NodeID,
		NetworkID:  local.NetworkID,
		Role:       local.Role,
		PublicKeys: local.PublicKeys,
		CreatedAt:  now,
	}
	if err := metadata.Validate(); err != nil {
		return tls.Certificate{}, err
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := randomSerial()
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   local.NodeID.String(),
			Organization: []string{"tailed-box-cli securemesh"},
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(certificateTTL),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		ExtraExtensions: []pkix.Extension{
			{
				Id:    metadataExtensionOID,
				Value: metadataBytes,
			},
		},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  privateKey,
		Leaf:        leaf,
	}, nil
}

func MetadataFromCertificate(certificate *x509.Certificate) (CertificateMetadata, error) {
	if certificate == nil {
		return CertificateMetadata{}, ErrMissingCertificate
	}
	var metadataBytes []byte
	for _, extension := range certificate.Extensions {
		if extension.Id.Equal(metadataExtensionOID) {
			metadataBytes = extension.Value
			break
		}
	}
	if len(metadataBytes) == 0 {
		return CertificateMetadata{}, ErrMissingMetadata
	}
	var metadata CertificateMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return CertificateMetadata{}, fmt.Errorf("%w: decode metadata: %w", ErrInvalidCertificate, err)
	}
	if err := metadata.Validate(); err != nil {
		return CertificateMetadata{}, err
	}
	certPublicKey, ok := certificate.PublicKey.(ed25519.PublicKey)
	if !ok {
		return CertificateMetadata{}, fmt.Errorf("%w: certificate public key is not ed25519", ErrInvalidCertificate)
	}
	if !bytes.Equal(certPublicKey, metadata.PublicKeys.Signing.Bytes) {
		return CertificateMetadata{}, fmt.Errorf("%w: certificate signing key does not match metadata", ErrInvalidCertificate)
	}
	return metadata, nil
}

type PeerLookup interface {
	Get(identity.NodeID) (peer.Record, error)
}

type RevocationChecker interface {
	IsRevoked(identity.NodeID) (bool, error)
}

type VerifyOptions struct {
	NetworkID    identity.NetworkID
	ExpectedRole identity.Role
	Peers        PeerLookup
	Revocations  RevocationChecker
	Now          func() time.Time
}

type Verifier struct {
	Options VerifyOptions
}

func (v Verifier) Verify(rawCerts [][]byte) (CertificateMetadata, error) {
	options := v.Options
	if err := options.Validate(); err != nil {
		return CertificateMetadata{}, err
	}
	if len(rawCerts) == 0 {
		return CertificateMetadata{}, ErrMissingCertificate
	}
	leaf, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return CertificateMetadata{}, fmt.Errorf("%w: parse certificate: %w", ErrInvalidCertificate, err)
	}
	return options.VerifyCertificate(leaf)
}

func (v Verifier) VerifyPeerCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	_, err := v.Verify(rawCerts)
	return err
}

func (o VerifyOptions) Validate() error {
	if err := o.NetworkID.Validate(); err != nil {
		return err
	}
	if o.ExpectedRole != "" {
		if err := o.ExpectedRole.Validate(); err != nil {
			return err
		}
	}
	if o.Peers == nil {
		return fmt.Errorf("%w: peer lookup is required", ErrInvalidOptions)
	}
	if o.Revocations == nil {
		return fmt.Errorf("%w: revocation checker is required", ErrInvalidOptions)
	}
	return nil
}

func (o VerifyOptions) VerifyCertificate(certificate *x509.Certificate) (CertificateMetadata, error) {
	if err := o.Validate(); err != nil {
		return CertificateMetadata{}, err
	}
	metadata, err := MetadataFromCertificate(certificate)
	if err != nil {
		return CertificateMetadata{}, err
	}
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	if current := now().UTC(); current.Before(certificate.NotBefore) || current.After(certificate.NotAfter) {
		return CertificateMetadata{}, fmt.Errorf("%w: certificate is outside validity window", ErrInvalidCertificate)
	}
	if metadata.NetworkID != o.NetworkID {
		return CertificateMetadata{}, ErrWrongNetwork
	}
	if o.ExpectedRole != "" && metadata.Role != o.ExpectedRole {
		return CertificateMetadata{}, ErrWrongRole
	}

	record, err := o.Peers.Get(metadata.NodeID)
	if err != nil {
		if errors.Is(err, peer.ErrPeerNotFound) {
			return CertificateMetadata{}, ErrUnknownPeer
		}
		return CertificateMetadata{}, err
	}
	if !record.Active() {
		return CertificateMetadata{}, ErrRevokedPeer
	}
	if record.NetworkID != o.NetworkID || record.NetworkID != metadata.NetworkID {
		return CertificateMetadata{}, ErrWrongNetwork
	}
	if record.Role != metadata.Role {
		return CertificateMetadata{}, ErrWrongRole
	}
	if !samePublicKeys(record.PublicKeys, metadata.PublicKeys) {
		return CertificateMetadata{}, ErrPublicKeyMismatch
	}
	revoked, err := o.Revocations.IsRevoked(metadata.NodeID)
	if err != nil {
		return CertificateMetadata{}, err
	}
	if revoked {
		return CertificateMetadata{}, ErrRevokedPeer
	}
	return metadata, nil
}

func NewTLSConfig(local identity.Identity, verifier Verifier, now time.Time) (*tls.Config, error) {
	certificate, err := NewCertificate(local, now)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:            tls.VersionTLS13,
		Certificates:          []tls.Certificate{certificate},
		NextProtos:            []string{ALPN},
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: verifier.VerifyPeerCertificate,
	}, nil
}

func signingPrivateKey(local identity.Identity) (ed25519.PrivateKey, error) {
	if local.PrivateKeys.Signing.Algorithm != identity.KeyAlgorithmEd25519 {
		return nil, fmt.Errorf("%w: signing private key must be ed25519", identity.ErrInvalidKey)
	}
	if len(local.PrivateKeys.Signing.Bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%w: invalid ed25519 private key size", identity.ErrInvalidKey)
	}
	privateKey := ed25519.PrivateKey(local.PrivateKeys.Signing.Bytes)
	return privateKey, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, err
	}
	return serial, nil
}

func samePublicKeys(a, b identity.PublicKeySet) bool {
	return a.Signing.Algorithm == b.Signing.Algorithm &&
		a.Transport.Algorithm == b.Transport.Algorithm &&
		bytes.Equal(a.Signing.Bytes, b.Signing.Bytes) &&
		bytes.Equal(a.Transport.Bytes, b.Transport.Bytes)
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

var _ RevocationChecker = revocation.Store{}
