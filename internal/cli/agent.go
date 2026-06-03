package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/tailedbox/tailedbox/internal/agent"
	"github.com/tailedbox/tailedbox/internal/logging"
)

func agentCommand() *command {
	parent := &command{
		name:        "agent",
		usage:       "tailedbox agent <command> [flags]",
		summary:     "Run and manage the local Tailedbox agent",
		description: "Run the lightweight local agent, inspect its health, and manage the Linux systemd service wrapper.",
	}
	attach(parent,
		&command{
			name:        "run",
			usage:       "tailedbox agent run [--heartbeat-interval 10s]",
			summary:     "Run the local agent in the foreground",
			description: "Run the lightweight Tailedbox agent loop in the foreground. Systemd uses this command as ExecStart.",
			needsConfig: true,
			run:         runAgentRun,
		},
		&command{
			name:        "status",
			usage:       "tailedbox agent status [--json]",
			summary:     "Show local agent health and memory usage",
			description: "Show the local agent heartbeat, uptime, and memory usage without exposing cluster inventory.",
			needsConfig: true,
			run:         runAgentStatus,
		},
		&command{
			name:        "install",
			usage:       "tailedbox agent install [--binary /usr/local/bin/tailedbox] [--unit-path /etc/systemd/system/tailedbox-agent.service] [--start] [--dry-run]",
			summary:     "Install the agent as a systemd service",
			description: "Write and enable a hardened systemd unit for the local Tailedbox agent. Use --dry-run to review the unit.",
			needsConfig: true,
			run:         runAgentInstall,
		},
		&command{
			name:        "uninstall",
			usage:       "tailedbox agent uninstall [--unit-path /etc/systemd/system/tailedbox-agent.service] [--dry-run]",
			summary:     "Remove the systemd service wrapper",
			description: "Disable, stop, and remove the Tailedbox systemd unit.",
			run:         runAgentUninstall,
		},
		&command{
			name:        "start",
			usage:       "tailedbox agent start",
			summary:     "Start the systemd agent service",
			description: "Start tailedbox-agent.service using systemctl.",
			run:         runAgentControl("start"),
		},
		&command{
			name:        "stop",
			usage:       "tailedbox agent stop",
			summary:     "Stop the systemd agent service",
			description: "Stop tailedbox-agent.service using systemctl.",
			run:         runAgentControl("stop"),
		},
		&command{
			name:        "restart",
			usage:       "tailedbox agent restart",
			summary:     "Restart the systemd agent service",
			description: "Restart tailedbox-agent.service using systemctl.",
			run:         runAgentControl("restart"),
		},
		&command{
			name:        "logs",
			usage:       "tailedbox agent logs [--follow] [--lines 100]",
			summary:     "Show local agent logs",
			description: "Show the same local redacted JSONL logs used by tailedbox logs.",
			needsConfig: true,
			run:         runAgentLogs,
		},
	)
	return parent
}

func runAgentRun(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("agent run", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	heartbeatInterval := fs.Duration("heartbeat-interval", agent.DefaultHeartbeatInterval, "How often the agent writes local health status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	fmt.Fprintln(a.stdout, a.theme.NoteLine("Tailedbox agent running. Press Ctrl+C to stop."))
	return agent.Run(ctx, a.cfg, agent.RunOptions{
		HeartbeatInterval: *heartbeatInterval,
		Logger:            a.logger,
	})
}

func runAgentStatus(_ context.Context, a *app, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	value, err := agent.ReadStatus(a.cfg, nowUTC())
	if err != nil {
		return err
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, value)
	}
	writeAgentStatus(a.stdout, a.theme, value)
	return nil
}

func runAgentInstall(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("agent install", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	binaryPath := fs.String("binary", "", "Path to the tailedbox binary for systemd ExecStart")
	unitPath := fs.String("unit-path", agent.DefaultSystemdUnitPath, "Systemd unit file path")
	dryRun := fs.Bool("dry-run", false, "Print the unit instead of writing or enabling it")
	start := fs.Bool("start", false, "Start the service after installing and enabling it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := agent.InstallSystemd(ctx, a.cfg, agent.ServiceOptions{
		BinaryPath: *binaryPath,
		UnitPath:   *unitPath,
		Start:      *start,
	}, *dryRun)
	if err != nil {
		return err
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, result)
	}
	if *dryRun {
		fmt.Fprintln(a.stdout, a.theme.Section("Systemd unit preview"))
		fmt.Fprintln(a.stdout)
		fmt.Fprint(a.stdout, result.Unit)
		return nil
	}
	fmt.Fprintln(a.stdout, a.theme.SuccessLine("Installed Tailedbox agent systemd service."))
	fmt.Fprintln(a.stdout)
	writeKeyValues(a.stdout, a.theme, "Service", [][2]string{
		{"Unit", result.UnitPath},
		{"Enabled on boot", a.theme.Bool(result.Enabled)},
		{"Started", a.theme.Bool(result.Started)},
	})
	return nil
}

func runAgentUninstall(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("agent uninstall", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	unitPath := fs.String("unit-path", agent.DefaultSystemdUnitPath, "Systemd unit file path")
	dryRun := fs.Bool("dry-run", false, "Show what would be removed without changing systemd")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := agent.UninstallSystemd(ctx, *unitPath, *dryRun)
	if err != nil {
		return err
	}
	if a.jsonOutput {
		return writeJSON(a.stdout, result)
	}
	if *dryRun {
		fmt.Fprintln(a.stdout, a.theme.NoteLine("Dry run: would disable, stop, and remove "+result.UnitPath+"."))
		return nil
	}
	fmt.Fprintln(a.stdout, a.theme.SuccessLine("Removed Tailedbox agent systemd service."))
	return nil
}

func runAgentControl(action string) runFunc {
	return func(ctx context.Context, a *app, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("unexpected argument %q", args[0])
		}
		if err := agent.ControlSystemd(ctx, action); err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, a.theme.SuccessLine("Systemd service "+action+" completed."))
		return nil
	}
}

func runAgentLogs(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("agent logs", flag.ContinueOnError)
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
}
