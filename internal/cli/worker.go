package cli

import (
	"context"
	"fmt"

	"github.com/tailedbox/tailedbox/internal/status"
)

func workerCommand() *command {
	worker := &command{
		name:        "worker",
		usage:       "tailedbox worker <command> [flags]",
		summary:     "Worker-node commands",
		description: "Manage the local worker role and inspect local-only worker state.",
	}
	attach(worker,
		&command{
			name:        "init",
			usage:       "tailedbox worker init [flags]",
			summary:     "Initialize this server as a worker",
			description: "Convenience alias for tailedbox init --role worker.",
			needsConfig: true,
			run:         runInitWithRole("worker"),
		},
		&command{
			name:        "status",
			usage:       "tailedbox worker status [--json]",
			summary:     "Show local worker-only status",
			description: "Show only this worker node's local state. Cluster inventory is intentionally not shown to workers in Part 1.",
			needsConfig: true,
			run: func(_ context.Context, a *app, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("unexpected argument %q", args[0])
				}
				value := status.ForWorker(a.cfg)
				if a.jsonOutput {
					return writeJSON(a.stdout, value)
				}
				writeWorkerStatus(a.stdout, a.theme, value)
				return nil
			},
		},
		plannedLeaf("join", "tailedbox worker join --code <join-code>", "Join a master cluster with a one-time code", "Worker enrollment"),
	)
	return worker
}
