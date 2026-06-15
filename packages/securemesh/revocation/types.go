package revocation

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

var ErrInvalidRecord = errors.New("invalid revocation record")

type Reason string

func (r Reason) String() string {
	return string(r)
}

func (r Reason) Empty() bool {
	return strings.TrimSpace(string(r)) == ""
}

type Record struct {
	NodeID    identity.NodeID `json:"node_id"`
	Role      identity.Role   `json:"role"`
	RevokedAt time.Time       `json:"revoked_at"`
	RevokedBy identity.NodeID `json:"revoked_by"`
	Reason    Reason          `json:"reason,omitempty"`
}

func (r Record) Validate() error {
	if err := r.NodeID.Validate(); err != nil {
		return err
	}
	if err := r.Role.Validate(); err != nil {
		return err
	}
	if r.RevokedAt.IsZero() {
		return fmt.Errorf("%w: revoked_at is required", ErrInvalidRecord)
	}
	if err := r.RevokedBy.Validate(); err != nil {
		return err
	}
	return nil
}
