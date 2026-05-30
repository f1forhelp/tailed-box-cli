package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/tailedbox/tailedbox/internal/logging"
)

func logsCommand() *command {
	return &command{
		name:        "logs",
		usage:       "tailedbox logs [--follow] [--lines 100]",
		summary:     "Show local Tailedbox logs",
		description: "Show local structured Tailedbox logs. Sensitive-looking values are redacted again before display.",
		needsConfig: true,
		run: func(ctx context.Context, a *app, args []string) error {
			fs := flag.NewFlagSet("logs", flag.ContinueOnError)
			fs.SetOutput(a.stderr)
			follow := fs.Bool("follow", false, "Follow log output")
			fs.BoolVar(follow, "f", false, "Follow log output")
			lines := fs.Int("lines", 100, "Number of recent lines to show")
			if err := fs.Parse(args); err != nil {
				return err
			}
			if fs.NArg() != 0 {
				return fmt.Errorf("unexpected argument %q", fs.Arg(0))
			}
			if err := logging.PrintLastLines(a.stdout, a.cfg.Paths.LogFile, *lines); err != nil {
				return err
			}
			if *follow {
				return logging.Follow(ctx, a.stdout, a.cfg.Paths.LogFile, time.Second)
			}
			return nil
		},
	}
}
