package actions

import (
	"context"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/network/pairing"
)

type PairingListener struct {
	EquivalentCLI string
	Bind          string
	Addr          string
	Serve         func(context.Context) error
	Close         func() error
}

func PreparePairingListener(ctx context.Context, bind string, options ...Option) (PairingListener, error) {
	if err := checkContext(ctx); err != nil {
		return PairingListener{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return PairingListener{}, err
	}
	if bind == "" {
		bind = pairing.DefaultBind
	}
	listener, err := pairing.NewServer(env.paths).Listen(bind)
	if err != nil {
		return PairingListener{}, err
	}
	return PairingListener{
		EquivalentCLI: Command("infra", "pair", "listen", "--bind", bind),
		Bind:          bind,
		Addr:          listener.Addr().String(),
		Serve:         listener.Serve,
		Close:         listener.Close,
	}, nil
}

func JoinPairing(ctx context.Context, endpoint, code string, role secureidentity.Role, masterNode secureidentity.NodeID, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	result, err := pairing.Join(ctx, env.paths, endpoint, code, role, masterNode)
	if err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "pair", "join", "--endpoint", endpoint, "--code", "<code>", "--role", role.String(), "--master-node", masterNode.String()),
		Message:       "pairing complete",
		Fields: map[string]string{
			"endpoint":       result.Endpoint,
			"local_node_id":  result.LocalNodeID.String(),
			"master_node_id": result.MasterNodeID.String(),
			"network_id":     result.NetworkID.String(),
			"role":           result.Role.String(),
		},
	}, nil
}
