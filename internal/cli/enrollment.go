package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tailedbox/tailedbox/internal/enrollment"
)

func runJoinWithRole(role string) runFunc {
	return func(ctx context.Context, a *app, args []string) error {
		fs := flag.NewFlagSet(role+" join", flag.ContinueOnError)
		fs.SetOutput(a.stderr)
		code := fs.String("code", "", "One-time join code from a master")
		masterStateDir := fs.String("master-state-dir", os.Getenv("TAILEDBOX_MASTER_STATE_DIR"), "Issuing master's state directory for the local Part 4 enrollment MVP")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("unexpected argument %q", fs.Arg(0))
		}
		result, err := enrollment.Join(a.cfg, enrollment.JoinOptions{
			ExpectedRole:   role,
			RawCode:        *code,
			MasterStateDir: *masterStateDir,
			Now:            time.Now(),
		})
		if err != nil {
			return err
		}
		if err := a.saveConfig(); err != nil {
			return err
		}
		a.logger.InfoContext(ctx, "node joined cluster", "role", role, "node_id", result.NodeID, "cluster_id", result.ClusterID, "join_code_id", result.JoinCodeID)
		if a.jsonOutput {
			return writeJSON(a.stdout, result)
		}
		fmt.Fprintln(a.stdout, a.theme.SuccessLine("Joined master cluster."))
		fmt.Fprintln(a.stdout)
		writeKeyValues(a.stdout, a.theme, "Cluster", [][2]string{
			{"Node ID", result.NodeID},
			{"Role", result.Role},
			{"Cluster ID", result.ClusterID},
			{"Cluster name", optionalString(result.ClusterName, "unnamed")},
			{"Master node", result.MasterNodeID},
			{"Trust anchor", result.MasterIdentityFingerprint},
			{"Join code ID", result.JoinCodeID},
			{"Reconnect lease expires", result.ReconnectLeaseExpiresAt.Format("2006-01-02 15:04:05 MST")},
		})
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, a.theme.NoteLine("Mesh connectivity and encrypted sessions arrive in the mesh POC parts."))
		return nil
	}
}
