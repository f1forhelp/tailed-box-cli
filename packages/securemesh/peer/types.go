package peer

import (
	"errors"
	"fmt"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

var ErrInvalidPeer = errors.New("invalid peer record")

type Status string

const (
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusRevoked:
		return true
	default:
		return false
	}
}

func (s Status) Validate() error {
	if !s.Valid() {
		return fmt.Errorf("%w: invalid status %q", ErrInvalidPeer, s)
	}
	return nil
}

type Endpoint struct {
	Address    string     `json:"address"`
	Protocol   string     `json:"protocol,omitempty"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

type Record struct {
	NodeID     identity.NodeID       `json:"node_id"`
	NetworkID  identity.NetworkID    `json:"network_id"`
	Role       identity.Role         `json:"role"`
	PublicKeys identity.PublicKeySet `json:"public_keys"`
	Endpoints  []Endpoint            `json:"endpoints,omitempty"`
	Status     Status                `json:"status"`
	AddedAt    time.Time             `json:"added_at"`
	RevokedAt  *time.Time            `json:"revoked_at,omitempty"`
	Metadata   map[string]string     `json:"metadata,omitempty"`
}

func (r Record) Active() bool {
	return r.Status == StatusActive && r.RevokedAt == nil
}

func (r Record) Validate() error {
	if err := r.NodeID.Validate(); err != nil {
		return err
	}
	if err := r.NetworkID.Validate(); err != nil {
		return err
	}
	if err := r.Role.Validate(); err != nil {
		return err
	}
	if err := r.PublicKeys.Validate(); err != nil {
		return err
	}
	if err := r.Status.Validate(); err != nil {
		return err
	}
	if r.AddedAt.IsZero() {
		return fmt.Errorf("%w: added_at is required", ErrInvalidPeer)
	}
	if r.Status == StatusRevoked && r.RevokedAt == nil {
		return fmt.Errorf("%w: revoked_at is required for revoked peers", ErrInvalidPeer)
	}
	return nil
}
