package actions

import (
	"context"
	"strconv"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
)

func ListPeers(ctx context.Context, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	records, err := peer.NewStore(env.paths).List()
	if err != nil {
		return Result{}, err
	}
	items := make([]map[string]string, 0, len(records))
	for _, record := range records {
		items = append(items, map[string]string{
			"node_id": record.NodeID.String(),
			"role":    record.Role.String(),
			"status":  string(record.Status),
		})
	}
	return Result{
		EquivalentCLI: Command("infra", "peer", "list"),
		Message:       "peers loaded",
		Fields: map[string]string{
			"count": strconv.Itoa(len(records)),
		},
		Items: items,
	}, nil
}
