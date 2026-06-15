package actions

import (
	"context"
	"errors"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

func RevokePeer(ctx context.Context, nodeID secureidentity.NodeID, role secureidentity.Role, reason string, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	identity, err := secureidentity.LoadIdentity(env.paths)
	if err != nil {
		return Result{}, err
	}
	if identity.Role != secureidentity.RoleMaster {
		return Result{}, ErrUnauthorized
	}

	revocationStore := revocation.NewStore(env.paths).WithClock(env.now)
	record, err := revocationStore.Revoke(nodeID, role, identity.NodeID, revocation.Reason(reason))
	if err != nil {
		return Result{}, err
	}
	if err := peer.NewStore(env.paths).MarkRevoked(record); err != nil && !errors.Is(err, peer.ErrPeerNotFound) {
		return Result{}, err
	}

	cmd := Command("infra", "peer", "revoke", "--node", nodeID.String(), "--role", role.String())
	if reason != "" {
		cmd = Command("infra", "peer", "revoke", "--node", nodeID.String(), "--role", role.String(), "--reason", reason)
	}
	return Result{
		EquivalentCLI: cmd,
		Message:       "peer revoked",
		Fields: map[string]string{
			"node_id":    record.NodeID.String(),
			"role":       record.Role.String(),
			"revoked_by": record.RevokedBy.String(),
		},
	}, nil
}
