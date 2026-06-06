package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/tailedbox/tailedbox/internal/buildinfo"
	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/logging"
	"github.com/tailedbox/tailedbox/internal/ui"
)

type app struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	build       buildinfo.Info
	theme       theme
	interactive bool

	configPath string
	stateDir   string
	logDir     string
	jsonOutput bool

	cfg      *config.Config
	logger   *slog.Logger
	closeLog func() error
}

func Execute(ctx context.Context, stdout, stderr io.Writer, args []string, build buildinfo.Info) error {
	a := &app{stdout: stdout, stderr: stderr, build: build, theme: newTheme(stdout)}
	return a.run(ctx, args)
}

func ExecuteInteractive(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, args []string, build buildinfo.Info) error {
	a := &app{
		stdin:       stdin,
		stdout:      stdout,
		stderr:      stderr,
		build:       build,
		theme:       newTheme(stdout),
		interactive: isInteractiveTerminal(stdin, stdout),
	}
	return a.run(ctx, args)
}

func (a *app) run(ctx context.Context, args []string) error {
	parsed, help, err := a.parseGlobalFlags(args)
	if err != nil {
		return err
	}

	if len(parsed) > 0 && parsed[0] == "help" {
		target, _ := rootCommand().find(parsed[1:])
		target.printHelp(a.stdout, a.theme)
		return nil
	}

	cmd, cmdArgs := rootCommand().find(parsed)
	if help {
		cmd.printHelp(a.stdout, a.theme)
		return nil
	}
	if len(parsed) == 0 && cmd.run == nil && a.interactive {
		return a.runInteractiveLoop(ctx)
	}
	if cmd.run == nil {
		cmd.printHelp(a.stdout, a.theme)
		return nil
	}

	if cmd.needsConfig {
		if err := a.initRuntime(); err != nil {
			return err
		}
		defer a.closeRuntime()
		a.logger.DebugContext(ctx, "command started", "command", cmd.path())
	}

	err = cmd.run(ctx, a, cmdArgs)
	if cmd.needsConfig {
		if err != nil {
			a.logger.ErrorContext(ctx, "command failed", "command", cmd.path(), "error", err)
		} else {
			a.logger.InfoContext(ctx, "command completed", "command", cmd.path())
		}
	}
	return err
}

func (a *app) runInteractiveLoop(ctx context.Context) error {
	var lastResult *ui.CommandResult
	for {
		selection, err := ui.Run(a.stdin, a.stdout, lastResult)
		if err != nil {
			return err
		}
		if selection.Quit || len(selection.Args) == 0 {
			return nil
		}
		lastResult = a.runInteractiveSelection(ctx, selection)
	}
}

func (a *app) runInteractiveSelection(ctx context.Context, selection ui.Selection) *ui.CommandResult {
	if selection.Action.EffectiveKind() == ui.ActionKindStream {
		return a.runInteractiveStream(ctx, selection)
	}
	var stdout, stderr bytes.Buffer
	commandApp := a.commandApp(&stdout, &stderr)
	err := commandApp.run(ctx, selection.Args)
	return commandResult(selection, stdout.String(), stderr.String(), err, false)
}

func (a *app) runInteractiveStream(ctx context.Context, selection ui.Selection) *ui.CommandResult {
	fmt.Fprintln(a.stdout)
	fmt.Fprintf(a.stdout, "Running %s. Press Ctrl+C to stop and return to the Tailedbox UI.\n\n", selection.Action.Title)

	streamCtx, stopStream := signal.NotifyContext(ctx, os.Interrupt)
	defer stopStream()

	commandApp := a.commandApp(a.stdout, a.stderr)
	err := commandApp.run(streamCtx, selection.Args)
	stopped := streamCtx.Err() != nil
	if stopped && errors.Is(err, context.Canceled) {
		err = nil
	}
	if stopped {
		fmt.Fprintln(a.stdout)
		fmt.Fprintln(a.stdout, "Stopped. Returning to the Tailedbox UI.")
	}
	return commandResult(selection, "", "", err, stopped)
}

func (a *app) commandApp(stdout, stderr io.Writer) *app {
	return &app{
		stdin:      a.stdin,
		stdout:     stdout,
		stderr:     stderr,
		build:      a.build,
		theme:      newTheme(stdout),
		configPath: a.configPath,
		stateDir:   a.stateDir,
		logDir:     a.logDir,
	}
}

func commandResult(selection ui.Selection, stdout, stderr string, err error, stopped bool) *ui.CommandResult {
	result := &ui.CommandResult{
		Title:   selection.Action.Title,
		Args:    append([]string(nil), selection.Args...),
		Stdout:  stdout,
		Stderr:  stderr,
		Kind:    selection.Action.EffectiveKind(),
		Stopped: stopped,
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func (a *app) initRuntime() error {
	if a.cfg != nil {
		return nil
	}
	cfg, err := config.Load(config.LoadOptions{
		ConfigPath: a.configPath,
		StateDir:   a.stateDir,
		LogDir:     a.logDir,
	})
	if err != nil {
		return err
	}
	logger, closeLog, err := logging.NewFileLogger(cfg.Paths.LogFile, cfg.Logging.DebugLogsEnabled)
	if err != nil {
		return fmt.Errorf("initialize logging: %w", err)
	}
	a.cfg = cfg
	a.logger = logger
	a.closeLog = closeLog
	return nil
}

func (a *app) closeRuntime() {
	if a.closeLog != nil {
		_ = a.closeLog()
	}
}

func (a *app) saveConfig() error {
	if err := config.Save(a.cfg); err != nil {
		return err
	}
	return nil
}

func (a *app) parseGlobalFlags(args []string) ([]string, bool, error) {
	parsed := make([]string, 0, len(args))
	help := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			parsed = append(parsed, args[i:]...)
			return parsed, help, nil
		case arg == "-h" || arg == "--help":
			help = true
		case arg == "--json":
			a.jsonOutput = true
		case arg == "--config" || arg == "--state-dir" || arg == "--log-dir":
			if i+1 >= len(args) {
				return nil, false, fmt.Errorf("%s requires a value", arg)
			}
			a.setGlobalFlag(arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--config="):
			a.configPath = strings.TrimPrefix(arg, "--config=")
		case strings.HasPrefix(arg, "--state-dir="):
			a.stateDir = strings.TrimPrefix(arg, "--state-dir=")
		case strings.HasPrefix(arg, "--log-dir="):
			a.logDir = strings.TrimPrefix(arg, "--log-dir=")
		default:
			parsed = append(parsed, arg)
		}
	}
	return parsed, help, nil
}

func (a *app) setGlobalFlag(name, value string) {
	switch name {
	case "--config":
		a.configPath = value
	case "--state-dir":
		a.stateDir = value
	case "--log-dir":
		a.logDir = value
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func isInteractiveTerminal(stdin io.Reader, stdout io.Writer) bool {
	in, inOK := stdin.(*os.File)
	out, outOK := stdout.(*os.File)
	if !inOK || !outOK {
		return false
	}
	inInfo, err := in.Stat()
	if err != nil {
		return false
	}
	outInfo, err := out.Stat()
	if err != nil {
		return false
	}
	return inInfo.Mode()&os.ModeCharDevice != 0 && outInfo.Mode()&os.ModeCharDevice != 0
}
