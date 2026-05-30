package cli

import (
	"context"
	"fmt"
)

func plannedLeaf(name, usage, summary, area string) *command {
	return &command{
		name:        name,
		usage:       usage,
		summary:     summary,
		description: plannedMessage(area),
		needsConfig: true,
		run: func(_ context.Context, _ *app, _ []string) error {
			return fmt.Errorf("%s", plannedMessage(area))
		},
	}
}
