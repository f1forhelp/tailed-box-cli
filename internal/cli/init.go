package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
)

func initCommand() *command {
	return &command{
		name:        "init",
		usage:       "tailedbox init --role master|worker [flags]",
		summary:     "Initialize this server as a master or worker",
		description: "Initialize the local Tailedbox role. Part 1 records non-secret local metadata only; node identity credentials arrive in Part 3.",
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
	changed, err := config.MarkInitialized(a.cfg, *role, time.Now())
	if err != nil {
		return err
	}
	if err := a.saveConfig(); err != nil {
		return err
	}

	a.logger.InfoContext(ctx, "node initialized", "role", a.cfg.Node.Role, "node_id", a.cfg.Node.ID, "changed", changed)
	if changed {
		fmt.Fprintln(a.stdout, a.theme.SuccessLine(fmt.Sprintf("Initialized tailedbox node as %s.", a.cfg.Node.Role)))
	} else {
		fmt.Fprintln(a.stdout, a.theme.NoteLine(fmt.Sprintf("Tailedbox node is already initialized as %s.", a.cfg.Node.Role)))
	}
	fmt.Fprintln(a.stdout)
	writeKeyValues(a.stdout, a.theme, "Local files", [][2]string{
		{"Node ID", a.cfg.Node.ID},
		{"Config file", a.cfg.Paths.ConfigFile},
		{"State dir", a.cfg.Paths.StateDir},
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
