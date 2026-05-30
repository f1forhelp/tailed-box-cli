package cli

import (
	"context"
	"fmt"

	"github.com/tailedbox/tailedbox/internal/status"
)

func masterCommand() *command {
	master := &command{
		name:        "master",
		usage:       "tailedbox master <command> [flags]",
		summary:     "Master/control-node commands",
		description: "Manage the local master/control-node role and inspect the cluster-aware master status shape.",
	}
	attach(master,
		&command{
			name:        "init",
			usage:       "tailedbox master init [flags]",
			summary:     "Initialize this server as a master",
			description: "Convenience alias for tailedbox init --role master.",
			needsConfig: true,
			run:         runInitWithRole("master"),
		},
		&command{
			name:        "status",
			usage:       "tailedbox master status [--json]",
			summary:     "Show current master and known cluster nodes",
			description: "Show the current master node plus the known master/worker inventory shape. Part 1 reports the local node only.",
			needsConfig: true,
			run: func(_ context.Context, a *app, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("unexpected argument %q", args[0])
				}
				value := status.ForMaster(a.cfg, nowUTC())
				if a.jsonOutput {
					return writeJSON(a.stdout, value)
				}
				writeMasterStatus(a.stdout, value)
				return nil
			},
		},
		plannedLeaf("join", "tailedbox master join --code <join-code>", "Join an existing master cluster", "Master join enrollment"),
		joinCodeCommand(),
	)
	return master
}

func joinCodeCommand() *command {
	parent := &command{
		name:        "join-code",
		usage:       "tailedbox master join-code <command> [flags]",
		summary:     "Manage one-time enrollment codes",
		description: "Join-code enrollment is planned for Part 4.",
	}
	attach(parent, plannedLeaf("create", "tailedbox master join-code create --role worker|master --ttl 15m", "Create a one-time enrollment code", "Join-code creation"))
	return parent
}
