package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tailedbox/link/control"
	"github.com/tailedbox/link/store"
	"github.com/tailedbox/tailedbox/internal/agent"
	"github.com/tailedbox/tailedbox/internal/config"
)

const uninstallConfirmation = "DELETE"

const debianPackageName = "tailedbox"

var installedBinaryPaths = []string{
	"/usr/bin/tailedbox",
	"/usr/local/bin/tailedbox",
}

type uninstallResult struct {
	DryRun               bool     `json:"dry_run"`
	Systemd              bool     `json:"systemd"`
	InstallArtifacts     bool     `json:"install_artifacts"`
	RequiresReinitialize bool     `json:"requires_reinitialize"`
	Removed              []string `json:"removed,omitempty"`
	WouldRemove          []string `json:"would_remove,omitempty"`
	Skipped              []string `json:"skipped,omitempty"`
	SystemdUnit          string   `json:"systemd_unit,omitempty"`
	PackageName          string   `json:"package_name,omitempty"`
}

type uninstallTarget struct {
	Path      string
	Kind      string
	Recursive bool
}

func uninstallCommand() *command {
	return &command{
		name:        "uninstall",
		usage:       "tailedbox uninstall [--dry-run] [--confirm-delete DELETE] [--systemd] [--install-artifacts] [--all] [--unit-path /etc/systemd/system/tailedbox-agent.service]",
		summary:     "Remove local Tailedbox identity and files",
		description: "Remove local Tailedbox config, state, logs, sockets, node identity, trust records, optional systemd unit, and optional installed package/binary artifacts.",
		run:         runUninstall,
	}
}

func runUninstall(ctx context.Context, a *app, args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	dryRun := fs.Bool("dry-run", false, "Show what would be removed without deleting files")
	confirm := fs.String("confirm-delete", "", "Required exact value DELETE to remove local files")
	includeSystemd := fs.Bool("systemd", false, "Also disable and remove the systemd service")
	includeInstallArtifacts := fs.Bool("install-artifacts", false, "Also purge the Debian package and remove known tailedbox command paths")
	includeAll := fs.Bool("all", false, "Remove local files, systemd service, Debian package, and known tailedbox command paths")
	unitPath := fs.String("unit-path", agent.DefaultSystemdUnitPath, "Systemd unit path to remove with --systemd")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if !*dryRun && *confirm != uninstallConfirmation {
		return fmt.Errorf("refusing to delete files without --confirm-delete %s", uninstallConfirmation)
	}
	if *includeAll {
		*includeSystemd = true
		*includeInstallArtifacts = true
	}

	paths, err := config.ResolvePaths(config.LoadOptions{
		ConfigPath: a.configPath,
		StateDir:   a.stateDir,
		LogDir:     a.logDir,
	})
	if err != nil {
		return err
	}
	targets, err := uninstallTargets(paths)
	if err != nil {
		return err
	}
	installTargets, err := installArtifactTargets()
	if err != nil {
		return err
	}

	result := uninstallResult{
		DryRun:               *dryRun,
		Systemd:              *includeSystemd,
		InstallArtifacts:     *includeInstallArtifacts,
		RequiresReinitialize: true,
	}
	if *includeSystemd {
		result.SystemdUnit = *unitPath
	}
	if *includeInstallArtifacts {
		result.PackageName = debianPackageName
	}
	if *dryRun {
		for _, target := range targets {
			result.WouldRemove = append(result.WouldRemove, target.Path)
		}
		if *includeSystemd {
			result.WouldRemove = append(result.WouldRemove, *unitPath)
		}
		if *includeInstallArtifacts {
			result.WouldRemove = append(result.WouldRemove, packageLabel(debianPackageName))
			for _, target := range installTargets {
				result.WouldRemove = append(result.WouldRemove, target.Path)
			}
		}
		return writeUninstallResult(a, result)
	}

	if *includeSystemd {
		if _, err := agent.UninstallSystemd(ctx, *unitPath, false); err != nil {
			return err
		}
		result.Removed = append(result.Removed, *unitPath)
	}

	if *includeInstallArtifacts {
		if err := removeInstallArtifacts(ctx, &result, installTargets); err != nil {
			return err
		}
	}

	for _, target := range targets {
		removed, err := removeUninstallTarget(target)
		if err != nil {
			return err
		}
		if removed {
			result.Removed = append(result.Removed, target.Path)
		} else {
			result.Skipped = append(result.Skipped, target.Path)
		}
	}
	removeEmptyConfigDir(paths.ConfigFile)
	return writeUninstallResult(a, result)
}

func installArtifactTargets() ([]uninstallTarget, error) {
	targets := make([]uninstallTarget, 0, len(installedBinaryPaths))
	for _, path := range installedBinaryPaths {
		clean, err := validateRemoveFile(path, "installed binary")
		if err != nil {
			return nil, err
		}
		targets = append(targets, uninstallTarget{Path: clean, Kind: "installed binary"})
	}
	return dedupeTargets(targets), nil
}

func removeInstallArtifacts(ctx context.Context, result *uninstallResult, targets []uninstallTarget) error {
	removedPackage, err := purgeDebianPackage(ctx, debianPackageName)
	if err != nil {
		return err
	}
	if removedPackage {
		result.Removed = append(result.Removed, packageLabel(debianPackageName))
	} else {
		result.Skipped = append(result.Skipped, packageLabel(debianPackageName))
	}

	for _, target := range targets {
		removed, err := removeUninstallTarget(target)
		if err != nil {
			return err
		}
		if removed {
			result.Removed = append(result.Removed, target.Path)
		} else {
			result.Skipped = append(result.Skipped, target.Path)
		}
	}
	return nil
}

func purgeDebianPackage(ctx context.Context, name string) (bool, error) {
	if runtime.GOOS != "linux" {
		return false, fmt.Errorf("install artifact removal is supported only on Linux, current OS is %s", runtime.GOOS)
	}
	if !commandAvailable("dpkg-query") {
		return false, nil
	}
	installed, err := debianPackageInstalled(ctx, name)
	if err != nil {
		return false, err
	}
	if !installed {
		return false, nil
	}
	switch {
	case commandAvailable("apt-get"):
		return true, runExternal(ctx, "apt-get", "purge", "-y", name)
	case commandAvailable("dpkg"):
		return true, runExternal(ctx, "dpkg", "--purge", name)
	default:
		return false, errors.New("apt-get or dpkg is required to purge the Debian package")
	}
}

func debianPackageInstalled(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Status}", name)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" || strings.Contains(text, "no packages found") {
			return false, nil
		}
		return false, fmt.Errorf("check Debian package %q: %w\n%s", name, err, text)
	}
	return strings.Contains(text, "install ok installed"), nil
}

func runExternal(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func packageLabel(name string) string {
	return "Debian package " + name
}

func uninstallTargets(paths config.Paths) ([]uninstallTarget, error) {
	var targets []uninstallTarget

	stateDir, err := validateRemoveDir(paths.StateDir, "state directory")
	if err != nil {
		return nil, err
	}
	logDir, err := validateRemoveDir(paths.LogDir, "log directory")
	if err != nil {
		return nil, err
	}
	configFile, err := validateRemoveFile(paths.ConfigFile, "config file")
	if err != nil {
		return nil, err
	}

	targets = append(targets, uninstallTarget{Path: configFile, Kind: "config file"})
	targets = append(targets, uninstallTarget{Path: stateDir, Kind: "state directory", Recursive: true})
	if !pathInsideOrEqual(logDir, stateDir) {
		targets = append(targets, uninstallTarget{Path: logDir, Kind: "log directory", Recursive: true})
	}

	socketPath := control.SocketPath(store.Paths{StateDir: paths.StateDir, AgentDir: paths.AgentDir})
	socketDir := filepath.Dir(socketPath)
	if !pathInsideOrEqual(socketDir, stateDir) {
		socketDir, err = validateRemoveDir(socketDir, "agent control socket fallback directory")
		if err != nil {
			return nil, err
		}
		targets = append(targets, uninstallTarget{Path: socketDir, Kind: "agent control socket fallback directory", Recursive: true})
	}
	return dedupeTargets(targets), nil
}

func removeUninstallTarget(target uninstallTarget) (bool, error) {
	if _, err := os.Lstat(target.Path); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("stat %s %q: %w", target.Kind, target.Path, err)
	}
	if target.Recursive {
		if err := os.RemoveAll(target.Path); err != nil {
			return false, fmt.Errorf("remove %s %q: %w", target.Kind, target.Path, err)
		}
		return true, nil
	}
	if err := os.Remove(target.Path); err != nil {
		return false, fmt.Errorf("remove %s %q: %w", target.Kind, target.Path, err)
	}
	return true, nil
}

func validateRemoveDir(path, label string) (string, error) {
	clean, err := cleanRemovePath(path, label)
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	for _, unsafe := range []string{home, os.TempDir()} {
		if unsafe == "" {
			continue
		}
		unsafe, err := filepath.Abs(filepath.Clean(unsafe))
		if err == nil && clean == unsafe {
			return "", fmt.Errorf("refusing to remove %s %q", label, clean)
		}
	}
	return clean, nil
}

func validateRemoveFile(path, label string) (string, error) {
	return cleanRemovePath(path, label)
}

func cleanRemovePath(path, label string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s path is empty", label)
	}
	if path == "." || path == ".." {
		return "", fmt.Errorf("refusing to remove %s %q", label, path)
	}
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", label, err)
	}
	if isVolumeRoot(clean) {
		return "", fmt.Errorf("refusing to remove %s %q", label, clean)
	}
	return clean, nil
}

func isVolumeRoot(path string) bool {
	return filepath.Clean(path) == filepath.VolumeName(path)+string(os.PathSeparator)
}

func pathInsideOrEqual(path, parent string) bool {
	pathAbs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	parentAbs, err := filepath.Abs(filepath.Clean(parent))
	if err != nil {
		return false
	}
	if pathAbs == parentAbs {
		return true
	}
	rel, err := filepath.Rel(parentAbs, pathAbs)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func dedupeTargets(targets []uninstallTarget) []uninstallTarget {
	seen := make(map[string]bool)
	deduped := make([]uninstallTarget, 0, len(targets))
	for _, target := range targets {
		if seen[target.Path] {
			continue
		}
		seen[target.Path] = true
		deduped = append(deduped, target)
	}
	return deduped
}

func removeEmptyConfigDir(configFile string) {
	dir := filepath.Dir(configFile)
	_ = os.Remove(dir)
}

func writeUninstallResult(a *app, result uninstallResult) error {
	if a.jsonOutput {
		return writeJSON(a.stdout, result)
	}
	if result.DryRun {
		fmt.Fprintln(a.stdout, a.theme.Warning("Dry run: no files were removed."))
		fmt.Fprintln(a.stdout)
		writePathList(a.stdout, a.theme, "Would remove", result.WouldRemove)
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, a.theme.NoteLine("This includes the local node identity, trust, enrollment, mesh, and agent state."))
		if result.InstallArtifacts {
			fmt.Fprintln(a.stdout, a.theme.NoteLine("This also includes the Debian package and known tailedbox command paths."))
		}
		fmt.Fprintln(a.stdout, a.theme.NoteLine("After uninstall, this system must be initialized again before Tailedbox can use it."))
		fmt.Fprintln(a.stdout)
		fmt.Fprintf(a.stdout, "Run with %s to delete these files.\n", a.theme.Command("--confirm-delete "+uninstallConfirmation))
		return nil
	}
	fmt.Fprintln(a.stdout, a.theme.SuccessLine("Removed local Tailedbox files."))
	fmt.Fprintln(a.stdout)
	writePathList(a.stdout, a.theme, "Removed", result.Removed)
	if len(result.Skipped) > 0 {
		fmt.Fprintln(a.stdout)
		writePathList(a.stdout, a.theme, "Already absent", result.Skipped)
	}
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, a.theme.NoteLine("The local node identity was removed. Run tailedbox init before using this system again."))
	if result.InstallArtifacts {
		fmt.Fprintln(a.stdout, a.theme.NoteLine("The installed package and known terminal command paths were also removed when present."))
	}
	return nil
}

func writePathList(w io.Writer, t theme, title string, paths []string) {
	fmt.Fprintln(w, t.Section(title))
	if len(paths) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}
	for _, path := range paths {
		fmt.Fprintf(w, "  %s\n", path)
	}
}
