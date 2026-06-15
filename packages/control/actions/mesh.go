package actions

import (
	"context"
	"fmt"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/network/tlstcp"
)

type MeshListener struct {
	EquivalentCLI string
	Bind          string
	Addr          string
	Serve         func(context.Context) error
	Close         func() error
}

func PrepareMeshListener(ctx context.Context, bind string, options ...Option) (MeshListener, error) {
	if err := checkContext(ctx); err != nil {
		return MeshListener{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return MeshListener{}, err
	}
	if bind == "" {
		bind = tlstcp.DefaultBind
	}
	listener, err := tlstcp.NewServer(env.paths).Listen(bind)
	if err != nil {
		return MeshListener{}, err
	}
	return MeshListener{
		EquivalentCLI: Command("infra", "mesh", "listen", "--bind", bind),
		Bind:          bind,
		Addr:          listener.Addr().String(),
		Serve:         listener.Serve,
		Close:         listener.Close,
	}, nil
}

func PingMesh(ctx context.Context, endpoint string, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	result, err := tlstcp.Ping(ctx, env.paths, endpoint)
	if err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "mesh", "ping", "--endpoint", endpoint),
		Message:       "mesh ping ok",
		Fields: map[string]string{
			"endpoint":       result.Endpoint,
			"local_node_id":  result.LocalNodeID.String(),
			"network_id":     result.NetworkID.String(),
			"remote_node_id": result.RemoteNodeID.String(),
			"remote_role":    result.RemoteRole.String(),
			"rtt":            fmt.Sprintf("%s", result.RTT),
		},
	}, nil
}
