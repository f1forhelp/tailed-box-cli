package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/f1forhelp/tailed-box-cli/packages/control/actions"
	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

type menuItem struct {
	Key           string
	Label         string
	EquivalentCLI string
	Run           func(context.Context, []actions.Option, *bufio.Scanner, io.Writer) (actions.Result, error)
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type actionSet struct {
	initNetwork     func(context.Context, ...actions.Option) (actions.Result, error)
	importNetwork   func(context.Context, secureidentity.NetworkID, ...actions.Option) (actions.Result, error)
	initIdentity    func(context.Context, secureidentity.Role, ...actions.Option) (actions.Result, error)
	showIdentity    func(context.Context, ...actions.Option) (actions.Result, error)
	createJoinCode  func(context.Context, secureidentity.Role, ...actions.Option) (actions.Result, error)
	consumeJoinCode func(context.Context, string, secureidentity.Role, ...actions.Option) (actions.Result, error)
	exportPeer      func(context.Context, ...actions.Option) (actions.Result, error)
	addPeer         func(context.Context, actions.PeerExport, ...actions.Option) (actions.Result, error)
	listPeers       func(context.Context, ...actions.Option) (actions.Result, error)
	revokePeer      func(context.Context, secureidentity.NodeID, secureidentity.Role, string, ...actions.Option) (actions.Result, error)
	prepareMesh     func(context.Context, string, ...actions.Option) (actions.MeshListener, error)
	pingMesh        func(context.Context, string, ...actions.Option) (actions.Result, error)
	preparePairing  func(context.Context, string, ...actions.Option) (actions.PairingListener, error)
	joinPairing     func(context.Context, string, string, secureidentity.Role, secureidentity.NodeID, ...actions.Option) (actions.Result, error)
}

var tuiActions = actionSet{
	initNetwork:     actions.InitNetwork,
	importNetwork:   actions.ImportNetwork,
	initIdentity:    actions.InitIdentity,
	showIdentity:    actions.ShowIdentity,
	createJoinCode:  actions.CreateJoinCode,
	consumeJoinCode: actions.ConsumeJoinCode,
	exportPeer:      actions.ExportPeer,
	addPeer:         actions.AddPeer,
	listPeers:       actions.ListPeers,
	revokePeer:      actions.RevokePeer,
	prepareMesh:     actions.PrepareMeshListener,
	pingMesh:        actions.PingMesh,
	preparePairing:  actions.PreparePairingListener,
	joinPairing:     actions.JoinPairing,
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	configRoot, ok := parseFlags(args, stderr)
	if !ok {
		return 2
	}
	options := actionOptions(configRoot)
	items := menuItems(tuiActions)

	fmt.Fprintln(stdout, "Secure Mesh TUI")
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
			result, err := item.Run(ctx, options, scanner, stdout)
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

func menuItems(actionSet actionSet) []menuItem {
	return []menuItem{
		{
			Key:           "1",
			Label:         "Initialize network",
			EquivalentCLI: actions.Command("infra", "network", "init"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.initNetwork(ctx, options...)
			},
		},
		{
			Key:           "2",
			Label:         "Import network",
			EquivalentCLI: actions.Command("infra", "network", "import", "--id", "<network_id>"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				networkID, err := prompt(scanner, stdout, "Network ID")
				if err != nil {
					return actions.Result{}, err
				}
				return actionSet.importNetwork(ctx, secureidentity.NetworkID(networkID), options...)
			},
		},
		{
			Key:           "3",
			Label:         "Initialize master identity",
			EquivalentCLI: actions.Command("infra", "identity", "init", "--role", "master"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.initIdentity(ctx, secureidentity.RoleMaster, options...)
			},
		},
		{
			Key:           "4",
			Label:         "Initialize worker identity",
			EquivalentCLI: actions.Command("infra", "identity", "init", "--role", "worker"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.initIdentity(ctx, secureidentity.RoleWorker, options...)
			},
		},
		{
			Key:           "5",
			Label:         "Show identity",
			EquivalentCLI: actions.Command("infra", "identity", "show"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.showIdentity(ctx, options...)
			},
		},
		{
			Key:           "6",
			Label:         "Create worker join code",
			EquivalentCLI: actions.Command("infra", "join-code", "create", "--role", "worker"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.createJoinCode(ctx, secureidentity.RoleWorker, options...)
			},
		},
		{
			Key:           "7",
			Label:         "Create master join code",
			EquivalentCLI: actions.Command("infra", "join-code", "create", "--role", "master"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.createJoinCode(ctx, secureidentity.RoleMaster, options...)
			},
		},
		{
			Key:           "8",
			Label:         "Consume local join code",
			EquivalentCLI: actions.Command("infra", "join-code", "consume", "--code", "<code>", "--role", "worker"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				code, role, err := promptCodeAndRole(scanner, stdout)
				if err != nil {
					return actions.Result{}, err
				}
				return actionSet.consumeJoinCode(ctx, code, role, options...)
			},
		},
		{
			Key:           "9",
			Label:         "Export public peer metadata",
			EquivalentCLI: actions.Command("infra", "peer", "export"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.exportPeer(ctx, options...)
			},
		},
		{
			Key:           "10",
			Label:         "Add peer from file",
			EquivalentCLI: actions.Command("infra", "peer", "add", "--file", "<peer.json>"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				filePath, err := prompt(scanner, stdout, "Peer file")
				if err != nil {
					return actions.Result{}, err
				}
				data, err := os.ReadFile(filePath)
				if err != nil {
					return actions.Result{}, err
				}
				exported, err := actions.DecodePeerExport(data)
				if err != nil {
					return actions.Result{}, err
				}
				return actionSet.addPeer(ctx, exported, options...)
			},
		},
		{
			Key:           "11",
			Label:         "List peers",
			EquivalentCLI: actions.Command("infra", "peer", "list"),
			Run: func(ctx context.Context, options []actions.Option, _ *bufio.Scanner, _ io.Writer) (actions.Result, error) {
				return actionSet.listPeers(ctx, options...)
			},
		},
		{
			Key:           "12",
			Label:         "Revoke peer",
			EquivalentCLI: actions.Command("infra", "peer", "revoke", "--node", "<node-id>", "--role", "worker", "--reason", "<reason>"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				nodeID, err := prompt(scanner, stdout, "Node ID")
				if err != nil {
					return actions.Result{}, err
				}
				role, err := promptRole(scanner, stdout)
				if err != nil {
					return actions.Result{}, err
				}
				reason, err := prompt(scanner, stdout, "Reason")
				if err != nil {
					return actions.Result{}, err
				}
				return actionSet.revokePeer(ctx, secureidentity.NodeID(nodeID), role, reason, options...)
			},
		},
		{
			Key:           "13",
			Label:         "Start mesh listener",
			EquivalentCLI: actions.Command("infra", "mesh", "listen", "--bind", "127.0.0.1:9443"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				bind, err := prompt(scanner, stdout, "Bind (empty for default)")
				if err != nil {
					return actions.Result{}, err
				}
				listener, err := actionSet.prepareMesh(ctx, bind, options...)
				if err != nil {
					return actions.Result{}, err
				}
				defer listener.Close()
				fmt.Fprintf(stdout, "mesh listener started\nbind: %s\naddress: %s\n", listener.Bind, listener.Addr)
				return actions.Result{}, listener.Serve(ctx)
			},
		},
		{
			Key:           "14",
			Label:         "Mesh ping",
			EquivalentCLI: actions.Command("infra", "mesh", "ping", "--endpoint", "<endpoint>"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				endpoint, err := prompt(scanner, stdout, "Endpoint")
				if err != nil {
					return actions.Result{}, err
				}
				return actionSet.pingMesh(ctx, endpoint, options...)
			},
		},
		{
			Key:           "15",
			Label:         "Start pairing listener",
			EquivalentCLI: actions.Command("infra", "pair", "listen", "--bind", "127.0.0.1:9444"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				bind, err := prompt(scanner, stdout, "Bind (empty for default)")
				if err != nil {
					return actions.Result{}, err
				}
				listener, err := actionSet.preparePairing(ctx, bind, options...)
				if err != nil {
					return actions.Result{}, err
				}
				defer listener.Close()
				fmt.Fprintf(stdout, "pairing listener started\nbind: %s\naddress: %s\n", listener.Bind, listener.Addr)
				return actions.Result{}, listener.Serve(ctx)
			},
		},
		{
			Key:           "16",
			Label:         "Join via pairing code",
			EquivalentCLI: actions.Command("infra", "pair", "join", "--endpoint", "<endpoint>", "--code", "<code>", "--role", "worker", "--master-node", "<node-id>"),
			Run: func(ctx context.Context, options []actions.Option, scanner *bufio.Scanner, stdout io.Writer) (actions.Result, error) {
				endpoint, err := prompt(scanner, stdout, "Endpoint")
				if err != nil {
					return actions.Result{}, err
				}
				code, role, err := promptCodeAndRole(scanner, stdout)
				if err != nil {
					return actions.Result{}, err
				}
				masterNode, err := prompt(scanner, stdout, "Master node ID")
				if err != nil {
					return actions.Result{}, err
				}
				return actionSet.joinPairing(ctx, endpoint, code, role, secureidentity.NodeID(masterNode), options...)
			},
		},
	}
}

func promptCodeAndRole(scanner *bufio.Scanner, stdout io.Writer) (string, secureidentity.Role, error) {
	code, err := prompt(scanner, stdout, "Join code")
	if err != nil {
		return "", "", err
	}
	role, err := promptRole(scanner, stdout)
	if err != nil {
		return "", "", err
	}
	return code, role, nil
}

func promptRole(scanner *bufio.Scanner, stdout io.Writer) (secureidentity.Role, error) {
	roleValue, err := prompt(scanner, stdout, "Role (master|worker)")
	if err != nil {
		return "", err
	}
	return secureidentity.ParseRole(roleValue)
}

func prompt(scanner *bufio.Scanner, stdout io.Writer, label string) (string, error) {
	fmt.Fprintf(stdout, "%s: ", label)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func printResult(stdout io.Writer, result actions.Result) {
	if result.RawOutput != "" {
		fmt.Fprint(stdout, result.RawOutput)
		return
	}
	if result.Message != "" {
		fmt.Fprintln(stdout, result.Message)
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
