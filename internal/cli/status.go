package cli

import (
	"context"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/status"
)

func statusCommand() *command {
	return &command{
		name:        "status",
		usage:       "tailedbox status [--json]",
		summary:     "Show local Tailedbox status based on the configured role",
		description: "Show local Tailedbox status. Master nodes use the cluster-aware shape; worker nodes show local-only state.",
		needsConfig: true,
		run: func(_ context.Context, a *app, _ []string) error {
			switch a.cfg.Node.Role {
			case config.RoleMaster:
				value := status.ForMaster(a.cfg, nowUTC())
				if a.jsonOutput {
					return writeJSON(a.stdout, value)
				}
				writeMasterStatus(a.stdout, a.theme, value)
			case config.RoleWorker:
				value := status.ForWorker(a.cfg)
				if a.jsonOutput {
					return writeJSON(a.stdout, value)
				}
				writeWorkerStatus(a.stdout, a.theme, value)
			default:
				value := status.ForLocal(a.cfg)
				if a.jsonOutput {
					return writeJSON(a.stdout, value)
				}
				writeLocalStatus(a.stdout, a.theme, value)
			}
			return nil
		},
	}
}
