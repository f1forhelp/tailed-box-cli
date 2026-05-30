package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type runFunc func(context.Context, *app, []string) error

type command struct {
	name        string
	usage       string
	summary     string
	description string
	parent      *command
	children    []*command
	run         runFunc
	needsConfig bool
}

func (c *command) find(args []string) (*command, []string) {
	if len(args) == 0 {
		return c, nil
	}
	for _, child := range c.children {
		if args[0] == child.name {
			return child.find(args[1:])
		}
	}
	return c, args
}

func (c *command) printHelp(w io.Writer) {
	if c.description != "" {
		fmt.Fprintln(w, c.description)
		fmt.Fprintln(w)
	}
	usage := c.usage
	if usage == "" {
		usage = c.path()
		if len(c.children) > 0 {
			usage += " <command> [flags]"
		} else {
			usage += " [flags]"
		}
	}
	fmt.Fprintf(w, "Usage:\n  %s\n", usage)

	if len(c.children) > 0 {
		fmt.Fprintln(w, "\nCommands:")
		width := 0
		for _, child := range c.children {
			if len(child.name) > width {
				width = len(child.name)
			}
		}
		for _, child := range c.children {
			fmt.Fprintf(w, "  %-*s  %s\n", width, child.name, child.summary)
		}
	}

	fmt.Fprintln(w, "\nGlobal Flags:")
	fmt.Fprintln(w, "  --config string      Config file path (default: XDG config path)")
	fmt.Fprintln(w, "  --state-dir string   State directory path (default: XDG state path)")
	fmt.Fprintln(w, "  --log-dir string     Log directory path (default: <state-dir>/logs)")
	fmt.Fprintln(w, "  --json               Emit JSON where supported")
	fmt.Fprintln(w, "  -h, --help           Show help")
}

func (c *command) path() string {
	var parts []string
	for current := c; current != nil; current = current.parent {
		parts = append(parts, current.name)
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, " ")
}

func attach(parent *command, children ...*command) {
	for _, child := range children {
		child.parent = parent
		parent.children = append(parent.children, child)
	}
}

func rootCommand() *command {
	root := &command{
		name:        "tailedbox",
		usage:       "tailedbox <command> [flags]",
		summary:     "Secure CLI-first VPS control system",
		description: "Tailedbox controls lightweight master/worker infrastructure from one binary.",
	}

	attach(root,
		versionCommand(),
		statusCommand(),
		initCommand(),
		masterCommand(),
		workerCommand(),
		logsCommand(),
		debugCommand(),
		meshCommand(),
		networkCommand(),
		nodeCommand(),
		pgCommand(),
	)

	return root
}
