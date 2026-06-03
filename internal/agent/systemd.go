package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tailedbox/tailedbox/internal/config"
)

const (
	SystemdServiceName     = "tailedbox-agent.service"
	DefaultSystemdUnitPath = "/etc/systemd/system/" + SystemdServiceName
)

type ServiceOptions struct {
	BinaryPath string
	UnitPath   string
	Start      bool
}

type InstallResult struct {
	UnitPath string `json:"unit_path"`
	Unit     string `json:"unit,omitempty"`
	Enabled  bool   `json:"enabled"`
	Started  bool   `json:"started"`
	DryRun   bool   `json:"dry_run"`
}

func RenderSystemdUnit(cfg *config.Config, opts ServiceOptions) (string, error) {
	if cfg == nil {
		return "", errors.New("config is nil")
	}
	if cfg.Node.ID == "" || cfg.Node.Role == "" {
		return "", errors.New("node must be initialized before installing the agent service")
	}
	binaryPath, err := resolveBinaryPath(opts.BinaryPath)
	if err != nil {
		return "", err
	}
	args := []string{
		binaryPath,
		"--config", cfg.Paths.ConfigFile,
		"--state-dir", cfg.Paths.StateDir,
		"--log-dir", cfg.Paths.LogDir,
		"agent", "run",
	}
	execStart := make([]string, 0, len(args))
	for _, arg := range args {
		execStart = append(execStart, systemdQuote(arg))
	}

	var b bytes.Buffer
	fmt.Fprintln(&b, "[Unit]")
	fmt.Fprintln(&b, "Description=Tailedbox lightweight agent")
	fmt.Fprintln(&b, "Documentation=https://github.com/tailedbox/tailedbox")
	fmt.Fprintln(&b, "After=network-online.target")
	fmt.Fprintln(&b, "Wants=network-online.target")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "[Service]")
	fmt.Fprintln(&b, "Type=simple")
	fmt.Fprintf(&b, "ExecStart=%s\n", strings.Join(execStart, " "))
	fmt.Fprintln(&b, "Restart=always")
	fmt.Fprintln(&b, "RestartSec=5s")
	fmt.Fprintln(&b, "KillSignal=SIGTERM")
	fmt.Fprintln(&b, "NoNewPrivileges=true")
	fmt.Fprintln(&b, "PrivateTmp=true")
	fmt.Fprintln(&b, "ProtectSystem=full")
	fmt.Fprintf(&b, "ReadWritePaths=%s %s %s\n", systemdQuote(filepath.Dir(cfg.Paths.ConfigFile)), systemdQuote(cfg.Paths.StateDir), systemdQuote(cfg.Paths.LogDir))
	fmt.Fprintln(&b, "StandardOutput=journal")
	fmt.Fprintln(&b, "StandardError=journal")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "[Install]")
	fmt.Fprintln(&b, "WantedBy=multi-user.target")
	return b.String(), nil
}

func InstallSystemd(ctx context.Context, cfg *config.Config, opts ServiceOptions, dryRun bool) (InstallResult, error) {
	unitPath := firstNonEmpty(opts.UnitPath, DefaultSystemdUnitPath)
	unit, err := RenderSystemdUnit(cfg, opts)
	if err != nil {
		return InstallResult{}, err
	}
	result := InstallResult{UnitPath: unitPath, Unit: unit, DryRun: dryRun}
	if dryRun {
		return result, nil
	}
	if runtime.GOOS != "linux" {
		return InstallResult{}, fmt.Errorf("systemd install is supported only on Linux, current OS is %s", runtime.GOOS)
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("create systemd unit directory: %w", err)
	}
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return InstallResult{}, fmt.Errorf("write systemd unit: %w", err)
	}
	if err := os.Chmod(unitPath, 0o644); err != nil {
		return InstallResult{}, fmt.Errorf("secure systemd unit permissions: %w", err)
	}
	if err := runSystemctl(ctx, "daemon-reload"); err != nil {
		return InstallResult{}, err
	}
	if err := runSystemctl(ctx, "enable", SystemdServiceName); err != nil {
		return InstallResult{}, err
	}
	result.Enabled = true
	if opts.Start {
		if err := runSystemctl(ctx, "start", SystemdServiceName); err != nil {
			return InstallResult{}, err
		}
		result.Started = true
	}
	result.Unit = ""
	return result, nil
}

func UninstallSystemd(ctx context.Context, unitPath string, dryRun bool) (InstallResult, error) {
	unitPath = firstNonEmpty(unitPath, DefaultSystemdUnitPath)
	result := InstallResult{UnitPath: unitPath, DryRun: dryRun}
	if dryRun {
		return result, nil
	}
	if runtime.GOOS != "linux" {
		return InstallResult{}, fmt.Errorf("systemd uninstall is supported only on Linux, current OS is %s", runtime.GOOS)
	}
	if err := runSystemctl(ctx, "disable", "--now", SystemdServiceName); err != nil {
		return InstallResult{}, err
	}
	if err := os.Remove(unitPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return InstallResult{}, fmt.Errorf("remove systemd unit: %w", err)
	}
	if err := runSystemctl(ctx, "daemon-reload"); err != nil {
		return InstallResult{}, err
	}
	return result, nil
}

func ControlSystemd(ctx context.Context, action string) error {
	switch action {
	case "start", "stop", "restart", "status":
	default:
		return fmt.Errorf("unsupported systemd action %q", action)
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("systemd %s is supported only on Linux, current OS is %s", action, runtime.GOOS)
	}
	return runSystemctl(ctx, action, SystemdServiceName)
}

func runSystemctl(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func resolveBinaryPath(path string) (string, error) {
	if path == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve current executable: %w", err)
		}
		path = executable
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve binary path: %w", err)
	}
	return absolute, nil
}

func systemdQuote(value string) string {
	if value == "" {
		return `""`
	}
	escaped := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`%`, `%%`,
		`$`, `$$`,
	).Replace(value)
	return `"` + escaped + `"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
