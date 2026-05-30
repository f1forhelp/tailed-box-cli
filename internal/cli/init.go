package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/nodeinit"
)

func initCommand() *command {
	return &command{
		name:        "init",
		usage:       "tailedbox init --role master|worker [flags]",
		summary:     "Initialize this server as a master or worker",
		description: "Initialize the local Tailedbox role, durable node identity, and agent-ready state files.",
		needsConfig: true,
		run:         runInit,
	}
}

func runInit(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	role := fs.String("role", "", "Node role: master or worker")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	initTime := time.Now()
	changed, err := config.MarkInitialized(a.cfg, *role, initTime)
	if err != nil {
		return err
	}
	initResult, err := nodeinit.Ensure(a.cfg, initTime)
	if err != nil {
		return err
	}
	if err := a.saveConfig(); err != nil {
		return err
	}

	a.logger.InfoContext(
		ctx,
		"node initialized",
		"role",
		a.cfg.Node.Role,
		"node_id",
		a.cfg.Node.ID,
		"changed",
		changed,
		"identity_created",
		initResult.IdentityCreated,
		"identity_fingerprint",
		initResult.IdentityFingerprint,
	)
	if changed {
		fmt.Fprintln(a.stdout, a.theme.SuccessLine(fmt.Sprintf("Initialized tailedbox node as %s.", a.cfg.Node.Role)))
	} else {
		fmt.Fprintln(a.stdout, a.theme.NoteLine(fmt.Sprintf("Tailedbox node is already initialized as %s.", a.cfg.Node.Role)))
	}
	fmt.Fprintln(a.stdout)
	writeKeyValues(a.stdout, a.theme, "Local files", [][2]string{
		{"Node ID", a.cfg.Node.ID},
		{"Identity fingerprint", initResult.IdentityFingerprint},
		{"Config file", a.cfg.Paths.ConfigFile},
		{"State dir", a.cfg.Paths.StateDir},
		{"Role dir", initResult.RoleDir},
		{"Public identity", initResult.PublicIdentityFile},
		{"Private identity", initResult.PrivateKeyFile},
		{"Agent config", initResult.AgentConfigFile},
		{"Log file", a.cfg.Paths.LogFile},
	})
	return nil
}

func runInitWithRole(role string) runFunc {
	return func(ctx context.Context, a *app, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("unexpected argument %q", args[0])
		}
		return runInit(ctx, a, []string{"--role", role})
	}
}
