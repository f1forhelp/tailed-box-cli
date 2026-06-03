package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/enrollment"
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
			description: "Show the current master node plus known trusted masters and workers.",
			needsConfig: true,
			run: func(_ context.Context, a *app, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("unexpected argument %q", args[0])
				}
				value := status.ForMaster(a.cfg, nowUTC())
				if a.jsonOutput {
					return writeJSON(a.stdout, value)
				}
				writeMasterStatus(a.stdout, a.theme, value)
				return nil
			},
		},
		&command{
			name:        "join",
			usage:       "tailedbox master join --code <join-code> --master-state-dir <path>",
			summary:     "Join an existing master cluster",
			description: "Join an existing master cluster with a one-time code. Until mesh transport exists, --master-state-dir points at the issuing master's local state.",
			needsConfig: true,
			run:         runJoinWithRole(config.RoleMaster),
		},
		joinCodeCommand(),
	)
	return master
}

func joinCodeCommand() *command {
	parent := &command{
		name:        "join-code",
		usage:       "tailedbox master join-code <command> [flags]",
		summary:     "Manage one-time enrollment codes",
		description: "Create one-time, short-lived enrollment codes for workers or additional masters.",
	}
	attach(parent, &command{
		name:        "create",
		usage:       "tailedbox master join-code create --role worker|master [--ttl 15m]",
		summary:     "Create a one-time enrollment code",
		description: "Create a short-lived, role-scoped join code. The raw code is printed once and only its hash is persisted.",
		needsConfig: true,
		run:         runJoinCodeCreate,
	})
	return parent
}

func runJoinCodeCreate(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("master join-code create", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	role := fs.String("role", config.RoleWorker, "Allowed joining role: worker or master")
	ttl := fs.Duration("ttl", enrollment.DefaultJoinCodeTTL, "Join code time to live")
	reconnectWindow := fs.Duration("reconnect-window", enrollment.DefaultReconnectWindow, "Master-controlled reconnect lease after enrollment")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := enrollment.CreateJoinCode(a.cfg, enrollment.CreateJoinCodeOptions{
		AllowedRole:     *role,
		TTL:             *ttl,
		ReconnectWindow: *reconnectWindow,
		Now:             time.Now(),
	})
	if err != nil {
		return err
	}
	a.logger.InfoContext(ctx, "join code created", "join_code_id", result.CodeID, "allowed_role", result.AllowedRole, "expires_at", result.ExpiresAt)
	if a.jsonOutput {
		return writeJSON(a.stdout, result)
	}
	fmt.Fprintln(a.stdout, a.theme.SuccessLine("Created one-time join code."))
	fmt.Fprintln(a.stdout)
	writeKeyValues(a.stdout, a.theme, "Enrollment", [][2]string{
		{"Join code", result.JoinCode},
		{"Code ID", result.CodeID},
		{"Allowed role", result.AllowedRole},
		{"Cluster ID", result.ClusterID},
		{"Trust anchor", result.IssuerFingerprint},
		{"Expires at", result.ExpiresAt.Format("2006-01-02 15:04:05 MST")},
		{"Reconnect window", result.ReconnectWindow},
		{"Master state dir", result.LocalMasterStateDir},
	})
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, a.theme.NoteLine("Until mesh enrollment exists, join with --master-state-dir "+result.LocalMasterStateDir+"."))
	return nil
}
