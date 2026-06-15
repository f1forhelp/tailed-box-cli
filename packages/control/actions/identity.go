package actions

import (
	"context"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func InitNetwork(ctx context.Context, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	network, err := secureidentity.GenerateNetwork(env.nowUTC(), "")
	if err != nil {
		return Result{}, err
	}
	if err := secureidentity.SaveNetwork(env.paths, network); err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "network", "init"),
		Message:       "network initialized",
		Fields: map[string]string{
			"network_id": network.ID.String(),
		},
	}, nil
}

func ImportNetwork(ctx context.Context, networkID secureidentity.NetworkID, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	if err := networkID.Validate(); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	network := secureidentity.Network{
		Version:   secureidentity.NetworkVersion,
		ID:        networkID,
		CreatedAt: env.nowUTC(),
	}
	if err := secureidentity.SaveNetwork(env.paths, network); err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "network", "import", "--id", networkID.String()),
		Message:       "network imported",
		Fields: map[string]string{
			"network_id": networkID.String(),
		},
	}, nil
}

func InitIdentity(ctx context.Context, role secureidentity.Role, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	network, err := secureidentity.LoadNetwork(env.paths)
	if err != nil {
		return Result{}, err
	}
	identity, err := secureidentity.GenerateIdentity(network.ID, role, env.nowUTC())
	if err != nil {
		return Result{}, err
	}
	if err := secureidentity.SaveIdentity(env.paths, identity); err != nil {
		return Result{}, err
	}
	if network.CreatedBy == "" {
		network.CreatedBy = identity.NodeID
		if err := secureidentity.SaveNetwork(env.paths, network); err != nil {
			return Result{}, err
		}
	}
	return Result{
		EquivalentCLI: Command("infra", "identity", "init", "--role", role.String()),
		Message:       "identity initialized",
		Fields: map[string]string{
			"node_id":    identity.NodeID.String(),
			"network_id": identity.NetworkID.String(),
			"role":       identity.Role.String(),
		},
	}, nil
}

func ShowIdentity(ctx context.Context, options ...Option) (Result, error) {
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
	return Result{
		EquivalentCLI: Command("infra", "identity", "show"),
		Message:       "identity loaded",
		Fields: map[string]string{
			"node_id":    identity.NodeID.String(),
			"network_id": identity.NetworkID.String(),
			"role":       identity.Role.String(),
		},
	}, nil
}
