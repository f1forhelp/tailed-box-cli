package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tailedbox/tailedbox/internal/buildinfo"
	"github.com/tailedbox/tailedbox/internal/cli"
)

func main() {
	signals := []os.Signal{syscall.SIGTERM}
	if hasCommandArg(os.Args[1:]) {
		signals = append(signals, os.Interrupt)
	}
	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop()

	if err := cli.ExecuteInteractive(ctx, os.Stdin, os.Stdout, os.Stderr, os.Args[1:], buildinfo.Current()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func hasCommandArg(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return i+1 < len(args)
		case arg == "--config" || arg == "--state-dir" || arg == "--log-dir":
			i++
		case arg == "-h" || arg == "--help" || arg == "--json":
		case strings.HasPrefix(arg, "--config="), strings.HasPrefix(arg, "--state-dir="), strings.HasPrefix(arg, "--log-dir="):
		default:
			return true
		}
	}
	return false
}
