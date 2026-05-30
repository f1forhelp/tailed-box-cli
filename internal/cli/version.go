package cli

import (
	"context"
	"fmt"
)

func versionCommand() *command {
	return &command{
		name:        "version",
		usage:       "tailedbox version [--json]",
		summary:     "Show build and runtime version information",
		description: "Show the Tailedbox build version, commit, build date, and Go runtime version.",
		run: func(_ context.Context, a *app, _ []string) error {
			if a.jsonOutput {
				return writeJSON(a.stdout, a.build)
			}
			fmt.Fprintf(a.stdout, "tailedbox %s\n", a.build.Version)
			fmt.Fprintf(a.stdout, "  commit: %s\n", a.build.Commit)
			fmt.Fprintf(a.stdout, "  built:  %s\n", a.build.Date)
			fmt.Fprintf(a.stdout, "  go:     %s\n", a.build.GoVersion)
			return nil
		},
	}
}
