package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tailedbox/tailedbox/internal/buildinfo"
	"github.com/tailedbox/tailedbox/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.ExecuteInteractive(ctx, os.Stdin, os.Stdout, os.Stderr, os.Args[1:], buildinfo.Current()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
