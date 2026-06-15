package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/f1forhelp/tailed-box-cli/packages/control/actions"
	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

type actionSet struct {
	initNetwork     func(context.Context, ...actions.Option) (actions.Result, error)
	initIdentity    func(context.Context, secureidentity.Role, ...actions.Option) (actions.Result, error)
	showIdentity    func(context.Context, ...actions.Option) (actions.Result, error)
	createJoinCode  func(context.Context, secureidentity.Role, ...actions.Option) (actions.Result, error)
	consumeJoinCode func(context.Context, string, secureidentity.Role, ...actions.Option) (actions.Result, error)
	listPeers       func(context.Context, ...actions.Option) (actions.Result, error)
	revokePeer      func(context.Context, secureidentity.NodeID, secureidentity.Role, string, ...actions.Option) (actions.Result, error)
}

var cliActions = actionSet{
	initNetwork:     actions.InitNetwork,
	initIdentity:    actions.InitIdentity,
	showIdentity:    actions.ShowIdentity,
	createJoinCode:  actions.CreateJoinCode,
	consumeJoinCode: actions.ConsumeJoinCode,
	listPeers:       actions.ListPeers,
	revokePeer:      actions.RevokePeer,
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	configRoot, remaining, ok := parseGlobalFlags(args, stderr)
	if !ok {
		return 2
	}
	if len(remaining) == 0 {
		usage(stderr)
		return 2
	}

	options := actionOptions(configRoot)
	result, err := dispatch(ctx, remaining, options, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printResult(stdout, result)
	return 0
}

func parseGlobalFlags(args []string, stderr io.Writer) (string, []string, bool) {
	flags := flag.NewFlagSet("infra", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configRoot := flags.String("config-root", "", "config root override")
	if err := flags.Parse(args); err != nil {
		return "", nil, false
	}
	return *configRoot, flags.Args(), true
}

func actionOptions(configRoot string) []actions.Option {
	if configRoot == "" {
		return nil
	}
	return []actions.Option{actions.WithConfigRoot(configRoot)}
}

func dispatch(ctx context.Context, args []string, options []actions.Option, stderr io.Writer) (actions.Result, error) {
	switch args[0] {
	case "network":
		return dispatchNetwork(ctx, args[1:], options, stderr)
	case "identity":
		return dispatchIdentity(ctx, args[1:], options, stderr)
	case "join-code":
		return dispatchJoinCode(ctx, args[1:], options, stderr)
	case "peer":
		return dispatchPeer(ctx, args[1:], options, stderr)
	default:
		return actions.Result{}, fmt.Errorf("unknown command %q", args[0])
	}
}

func dispatchNetwork(ctx context.Context, args []string, options []actions.Option, stderr io.Writer) (actions.Result, error) {
	if len(args) == 1 && args[0] == "init" {
		return cliActions.initNetwork(ctx, options...)
	}
	return actions.Result{}, fmt.Errorf("usage: infra network init")
}

func dispatchIdentity(ctx context.Context, args []string, options []actions.Option, stderr io.Writer) (actions.Result, error) {
	if len(args) == 1 && args[0] == "show" {
		return cliActions.showIdentity(ctx, options...)
	}
	if len(args) == 0 || args[0] != "init" {
		return actions.Result{}, fmt.Errorf("usage: infra identity init --role master|worker")
	}
	flags := flag.NewFlagSet("identity init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	roleValue := flags.String("role", "", "node role")
	if err := flags.Parse(args[1:]); err != nil {
		return actions.Result{}, err
	}
	role, err := secureidentity.ParseRole(*roleValue)
	if err != nil {
		return actions.Result{}, err
	}
	return cliActions.initIdentity(ctx, role, options...)
}

func dispatchJoinCode(ctx context.Context, args []string, options []actions.Option, stderr io.Writer) (actions.Result, error) {
	if len(args) == 0 {
		return actions.Result{}, fmt.Errorf("usage: infra join-code create|consume")
	}
	switch args[0] {
	case "create":
		flags := flag.NewFlagSet("join-code create", flag.ContinueOnError)
		flags.SetOutput(stderr)
		roleValue := flags.String("role", "", "joining node role")
		if err := flags.Parse(args[1:]); err != nil {
			return actions.Result{}, err
		}
		role, err := secureidentity.ParseRole(*roleValue)
		if err != nil {
			return actions.Result{}, err
		}
		return cliActions.createJoinCode(ctx, role, options...)
	case "consume":
		flags := flag.NewFlagSet("join-code consume", flag.ContinueOnError)
		flags.SetOutput(stderr)
		code := flags.String("code", "", "join code")
		roleValue := flags.String("role", "", "joining node role")
		if err := flags.Parse(args[1:]); err != nil {
			return actions.Result{}, err
		}
		role, err := secureidentity.ParseRole(*roleValue)
		if err != nil {
			return actions.Result{}, err
		}
		return cliActions.consumeJoinCode(ctx, *code, role, options...)
	default:
		return actions.Result{}, fmt.Errorf("usage: infra join-code create|consume")
	}
}

func dispatchPeer(ctx context.Context, args []string, options []actions.Option, stderr io.Writer) (actions.Result, error) {
	if len(args) == 1 && args[0] == "list" {
		return cliActions.listPeers(ctx, options...)
	}
	if len(args) == 0 || args[0] != "revoke" {
		return actions.Result{}, fmt.Errorf("usage: infra peer list|revoke")
	}
	flags := flag.NewFlagSet("peer revoke", flag.ContinueOnError)
	flags.SetOutput(stderr)
	nodeID := flags.String("node", "", "node id")
	roleValue := flags.String("role", "", "node role")
	reason := flags.String("reason", "", "revocation reason")
	if err := flags.Parse(args[1:]); err != nil {
		return actions.Result{}, err
	}
	role, err := secureidentity.ParseRole(*roleValue)
	if err != nil {
		return actions.Result{}, err
	}
	return cliActions.revokePeer(ctx, secureidentity.NodeID(*nodeID), role, *reason, options...)
}

func printResult(stdout io.Writer, result actions.Result) {
	if result.Message != "" {
		fmt.Fprintln(stdout, result.Message)
	}
	if result.EquivalentCLI != "" {
		fmt.Fprintf(stdout, "equivalent CLI: %s\n", result.EquivalentCLI)
	}
	if len(result.Fields) > 0 {
		keys := make([]string, 0, len(result.Fields))
		for key := range result.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(stdout, "%s: %s\n", key, result.Fields[key])
		}
	}
	for _, item := range result.Items {
		keys := make([]string, 0, len(item))
		for key := range item {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for idx, key := range keys {
			if idx > 0 {
				fmt.Fprint(stdout, " ")
			}
			fmt.Fprintf(stdout, "%s=%s", key, item[key])
		}
		fmt.Fprintln(stdout)
	}
	if result.SecretLabel != "" && result.SecretValue != "" {
		fmt.Fprintf(stdout, "%s: %s\n", result.SecretLabel, result.SecretValue)
	}
}

func usage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: infra [--config-root path] <command>")
	fmt.Fprintln(stderr, "commands: network init, identity init, identity show, join-code create, join-code consume, peer list, peer revoke")
}
