package cli

import (
	"context"
	"fmt"
	"strings"
)

func debugCommand() *command {
	debug := &command{
		name:        "debug",
		usage:       "tailedbox debug <command> [flags]",
		summary:     "Debug and troubleshooting controls",
		description: "Manage opt-in diagnostics. Deep debug logs are disabled by default and still pass through redaction.",
	}
	logs := &command{
		name:        "logs",
		usage:       "tailedbox debug logs <enable|disable>",
		summary:     "Enable or disable deep debug logs",
		description: "Toggle deep debug logs for troubleshooting. Debug logs remain redacted and should not include secrets or decrypted payloads.",
	}
	attach(logs,
		debugLogsToggleCommand("enable", true),
		debugLogsToggleCommand("disable", false),
	)
	attach(debug, logs)
	return debug
}

func debugLogsToggleCommand(name string, enabled bool) *command {
	return &command{
		name:        name,
		usage:       "tailedbox debug logs " + name,
		summary:     fmt.Sprintf("%s deep debug logs", titleCase(name)),
		needsConfig: true,
		run: func(ctx context.Context, a *app, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("unexpected argument %q", args[0])
			}
			a.cfg.Logging.DebugLogsEnabled = enabled
			if err := a.saveConfig(); err != nil {
				return err
			}
			a.logger.InfoContext(ctx, "debug log setting changed", "enabled", enabled)
			if enabled {
				fmt.Fprintln(a.stdout, a.theme.SuccessLine("Deep debug logs enabled."))
				fmt.Fprintln(a.stdout, a.theme.NoteLine("Sensitive-looking values will continue to be redacted."))
			} else {
				fmt.Fprintln(a.stdout, a.theme.SuccessLine("Deep debug logs disabled."))
			}
			return nil
		},
	}
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
