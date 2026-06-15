package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/f1forhelp/tailed-box-cli/packages/control/actions"
	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

type menuItem struct {
	Key           string
	Label         string
	EquivalentCLI string
	Run           func(context.Context, []actions.Option) (actions.Result, error)
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	configRoot, ok := parseFlags(args, stderr)
	if !ok {
		return 2
	}
	options := actionOptions(configRoot)
	items := menuItems()

	fmt.Fprintln(stdout, "Secure Mesh Foundation TUI")
	for _, item := range items {
		fmt.Fprintf(stdout, "%s) %s\n", item.Key, item.Label)
		fmt.Fprintf(stdout, "   CLI: %s\n", item.EquivalentCLI)
	}
	fmt.Fprintln(stdout, "q) Quit")
	fmt.Fprint(stdout, "Select: ")

	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}
	choice := strings.TrimSpace(scanner.Text())
	if choice == "q" || choice == "quit" {
		return 0
	}
	for _, item := range items {
		if item.Key == choice {
			fmt.Fprintf(stdout, "equivalent CLI: %s\n", item.EquivalentCLI)
			result, err := item.Run(ctx, options)
			if err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				return 1
			}
			printResult(stdout, result)
			return 0
		}
	}
	fmt.Fprintf(stderr, "error: unknown selection %q\n", choice)
	return 2
}

func parseFlags(args []string, stderr io.Writer) (string, bool) {
	flags := flag.NewFlagSet("infra-tui", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configRoot := flags.String("config-root", "", "config root override")
	if err := flags.Parse(args); err != nil {
		return "", false
	}
	return *configRoot, true
}

func actionOptions(configRoot string) []actions.Option {
	if configRoot == "" {
		return nil
	}
	return []actions.Option{actions.WithConfigRoot(configRoot)}
}

func menuItems() []menuItem {
	return []menuItem{
		{
			Key:           "1",
			Label:         "Create worker join code",
			EquivalentCLI: actions.Command("infra", "join-code", "create", "--role", "worker"),
			Run: func(ctx context.Context, options []actions.Option) (actions.Result, error) {
				return actions.CreateJoinCode(ctx, secureidentity.RoleWorker, options...)
			},
		},
		{
			Key:           "2",
			Label:         "List peers",
			EquivalentCLI: actions.Command("infra", "peer", "list"),
			Run: func(ctx context.Context, options []actions.Option) (actions.Result, error) {
				return actions.ListPeers(ctx, options...)
			},
		},
		{
			Key:           "3",
			Label:         "Show identity",
			EquivalentCLI: actions.Command("infra", "identity", "show"),
			Run: func(ctx context.Context, options []actions.Option) (actions.Result, error) {
				return actions.ShowIdentity(ctx, options...)
			},
		},
	}
}

func printResult(stdout io.Writer, result actions.Result) {
	if result.Message != "" {
		fmt.Fprintln(stdout, result.Message)
	}
	if len(result.Fields) > 0 {
		for key, value := range result.Fields {
			fmt.Fprintf(stdout, "%s: %s\n", key, value)
		}
	}
	for _, item := range result.Items {
		for key, value := range item {
			fmt.Fprintf(stdout, "%s: %s ", key, value)
		}
		fmt.Fprintln(stdout)
	}
	if result.SecretLabel != "" && result.SecretValue != "" {
		fmt.Fprintf(stdout, "%s: %s\n", result.SecretLabel, result.SecretValue)
	}
}
