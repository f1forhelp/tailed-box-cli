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
			fmt.Fprintf(a.stdout, "%s %s\n\n", a.theme.Title("tailedbox"), a.theme.Accent(a.build.Version))
			writeKeyValues(a.stdout, a.theme, "Build", [][2]string{
				{"Commit", a.build.Commit},
				{"Built", a.build.Date},
				{"Go", a.build.GoVersion},
			})
			return nil
		},
	}
}
