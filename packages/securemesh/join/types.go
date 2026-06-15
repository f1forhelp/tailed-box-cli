package join

import (
	"errors"
	"fmt"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

var (
	ErrInvalidCodeID = errors.New("invalid join code id")
	ErrInvalidRecord = errors.New("invalid join code record")
)

type CodeID string

func (id CodeID) String() string {
	return string(id)
}

func (id CodeID) Validate() error {
	if id == "" {
		return ErrInvalidCodeID
	}
	return nil
}

type Status string

const (
	StatusUnused   Status = "unused"
	StatusConsumed Status = "consumed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusUnused, StatusConsumed:
		return true
	default:
		return false
	}
}

func (s Status) Validate() error {
	if !s.Valid() {
		return fmt.Errorf("%w: invalid status %q", ErrInvalidRecord, s)
	}
	return nil
}

type Record struct {
	ID                CodeID             `json:"id"`
	NetworkID         identity.NetworkID `json:"network_id"`
	ExpectedRole      identity.Role      `json:"expected_role"`
	IssuedBy          identity.NodeID    `json:"issued_by"`
	CreatedAt         time.Time          `json:"created_at"`
	VerifierAlgorithm string             `json:"verifier_algorithm"`
	Salt              []byte             `json:"salt"`
	Verifier          []byte             `json:"verifier"`
	Status            Status             `json:"status"`
	ConsumedAt        *time.Time         `json:"consumed_at,omitempty"`
	ConsumedBy        identity.NodeID    `json:"consumed_by,omitempty"`
}

func (r Record) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.NetworkID.Validate(); err != nil {
		return err
	}
	if err := r.ExpectedRole.Validate(); err != nil {
		return err
	}
	if err := r.IssuedBy.Validate(); err != nil {
		return err
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_at is required", ErrInvalidRecord)
	}
	if r.VerifierAlgorithm == "" {
		return fmt.Errorf("%w: verifier algorithm is required", ErrInvalidRecord)
	}
	if len(r.Salt) == 0 {
		return fmt.Errorf("%w: salt is required", ErrInvalidRecord)
	}
	if len(r.Verifier) == 0 {
		return fmt.Errorf("%w: verifier is required", ErrInvalidRecord)
	}
	if err := r.Status.Validate(); err != nil {
		return err
	}
	if r.Status == StatusConsumed {
		if r.ConsumedAt == nil || r.ConsumedAt.IsZero() {
			return fmt.Errorf("%w: consumed_at is required for consumed codes", ErrInvalidRecord)
		}
		if err := r.ConsumedBy.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type CreateRequest struct {
	NetworkID    identity.NetworkID `json:"network_id"`
	ExpectedRole identity.Role      `json:"expected_role"`
	IssuedBy     identity.NodeID    `json:"issued_by"`
}

type ConsumeRequest struct {
	Code         string             `json:"-"`
	NetworkID    identity.NetworkID `json:"network_id"`
	ExpectedRole identity.Role      `json:"expected_role"`
	ConsumedBy   identity.NodeID    `json:"consumed_by"`
}

type ConsumeResult struct {
	Record   Record `json:"record"`
	Consumed bool   `json:"consumed"`
}
