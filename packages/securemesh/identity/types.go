package identity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidNodeID    = errors.New("invalid node id")
	ErrInvalidNetworkID = errors.New("invalid network id")
	ErrInvalidRole      = errors.New("invalid node role")
	ErrInvalidKey       = errors.New("invalid key material")
)

type NodeID string

func (id NodeID) String() string {
	return string(id)
}

func (id NodeID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidNodeID
	}
	return nil
}

type NetworkID string

func (id NetworkID) String() string {
	return string(id)
}

func (id NetworkID) Validate() error {
	if strings.TrimSpace(string(id)) == "" {
		return ErrInvalidNetworkID
	}
	return nil
}

type Role string

const (
	RoleMaster Role = "master"
	RoleWorker Role = "worker"
)

func ParseRole(value string) (Role, error) {
	role := Role(strings.ToLower(strings.TrimSpace(value)))
	if err := role.Validate(); err != nil {
		return "", err
	}
	return role, nil
}

func (r Role) String() string {
	return string(r)
}

func (r Role) Valid() bool {
	switch r {
	case RoleMaster, RoleWorker:
		return true
	default:
		return false
	}
}

func (r Role) Validate() error {
	if !r.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidRole, r)
	}
	return nil
}

type KeyAlgorithm string

const (
	KeyAlgorithmEd25519 KeyAlgorithm = "ed25519"
	KeyAlgorithmX25519  KeyAlgorithm = "x25519"
)

func (a KeyAlgorithm) String() string {
	return string(a)
}

func (a KeyAlgorithm) Valid() bool {
	switch a {
	case KeyAlgorithmEd25519, KeyAlgorithmX25519:
		return true
	default:
		return false
	}
}

func (a KeyAlgorithm) Validate() error {
	if !a.Valid() {
		return fmt.Errorf("%w: unsupported algorithm %q", ErrInvalidKey, a)
	}
	return nil
}

type PublicKey struct {
	Algorithm KeyAlgorithm `json:"algorithm"`
	Bytes     []byte       `json:"bytes"`
}

func (k PublicKey) Validate() error {
	if err := k.Algorithm.Validate(); err != nil {
		return err
	}
	if len(k.Bytes) == 0 {
		return fmt.Errorf("%w: empty public key", ErrInvalidKey)
	}
	return nil
}

type PrivateKey struct {
	Algorithm KeyAlgorithm `json:"algorithm"`
	Bytes     []byte       `json:"bytes"`
}

func (k PrivateKey) Validate() error {
	if err := k.Algorithm.Validate(); err != nil {
		return err
	}
	if len(k.Bytes) == 0 {
		return fmt.Errorf("%w: empty private key", ErrInvalidKey)
	}
	return nil
}

type PublicKeySet struct {
	Signing   PublicKey `json:"signing"`
	Transport PublicKey `json:"transport"`
}

func (s PublicKeySet) Validate() error {
	if err := s.Signing.Validate(); err != nil {
		return err
	}
	if s.Signing.Algorithm != KeyAlgorithmEd25519 {
		return fmt.Errorf("%w: signing key must be ed25519", ErrInvalidKey)
	}
	if err := s.Transport.Validate(); err != nil {
		return err
	}
	if s.Transport.Algorithm != KeyAlgorithmX25519 {
		return fmt.Errorf("%w: transport key must be x25519", ErrInvalidKey)
	}
	return nil
}

type PrivateKeySet struct {
	Signing   PrivateKey `json:"signing"`
	Transport PrivateKey `json:"transport"`
}

func (s PrivateKeySet) Validate() error {
	if err := s.Signing.Validate(); err != nil {
		return err
	}
	if s.Signing.Algorithm != KeyAlgorithmEd25519 {
		return fmt.Errorf("%w: signing key must be ed25519", ErrInvalidKey)
	}
	if err := s.Transport.Validate(); err != nil {
		return err
	}
	if s.Transport.Algorithm != KeyAlgorithmX25519 {
		return fmt.Errorf("%w: transport key must be x25519", ErrInvalidKey)
	}
	return nil
}

type Identity struct {
	Version     int           `json:"version"`
	NodeID      NodeID        `json:"node_id"`
	NetworkID   NetworkID     `json:"network_id"`
	Role        Role          `json:"role"`
	PublicKeys  PublicKeySet  `json:"public_keys"`
	PrivateKeys PrivateKeySet `json:"private_keys"`
	CreatedAt   time.Time     `json:"created_at"`
}

func (i Identity) Validate() error {
	if i.Version <= 0 {
		return errors.New("identity version must be positive")
	}
	if err := i.NodeID.Validate(); err != nil {
		return err
	}
	if err := i.NetworkID.Validate(); err != nil {
		return err
	}
	if err := i.Role.Validate(); err != nil {
		return err
	}
	if err := i.PublicKeys.Validate(); err != nil {
		return err
	}
	if err := i.PrivateKeys.Validate(); err != nil {
		return err
	}
	if i.CreatedAt.IsZero() {
		return errors.New("identity created_at is required")
	}
	return nil
}

type Network struct {
	Version   int       `json:"version"`
	ID        NetworkID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy NodeID    `json:"created_by,omitempty"`
}

func (n Network) Validate() error {
	if n.Version <= 0 {
		return errors.New("network version must be positive")
	}
	if err := n.ID.Validate(); err != nil {
		return err
	}
	if n.CreatedAt.IsZero() {
		return errors.New("network created_at is required")
	}
	return nil
}
