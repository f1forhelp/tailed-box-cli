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
	group       string
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

func (c *command) printHelp(w io.Writer, t theme) {
	fmt.Fprintln(w, t.Title(c.path()))
	if c.description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  "+c.description)
	}
	if c.parent == nil {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  "+t.Subtitle("Secure, lightweight VPS control from one Go binary."))
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
	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Section("Usage"))
	fmt.Fprintf(w, "  %s\n", t.Accent(usage))

	if len(c.children) > 0 {
		printCommandGroups(w, t, c.children)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Section("Global Flags"))
	printRows(w, t, []row{
		{"--config string", "Config file path (default: XDG config path)"},
		{"--state-dir string", "State directory path (default: XDG state path)"},
		{"--log-dir string", "Log directory path (default: <state-dir>/logs)"},
		{"--json", "Emit JSON where supported"},
		{"-h, --help", "Show help"},
	})
}

type row struct {
	left  string
	right string
}

func printCommandGroups(w io.Writer, t theme, children []*command) {
	groups := orderedGroups(children)
	for _, group := range groups {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.Section(group))
		var rows []row
		for _, child := range children {
			if commandGroup(child) != group {
				continue
			}
			rows = append(rows, row{child.name, child.summary})
		}
		printRows(w, t, rows)
	}
}

func printRows(w io.Writer, t theme, rows []row) {
	width := 0
	for _, row := range rows {
		if rowWidth := t.Width(row.left); rowWidth > width {
			width = rowWidth
		}
	}
	for _, row := range rows {
		fmt.Fprintf(w, "  %s  %s\n", t.Command(padRight(t, row.left, width)), row.right)
	}
}

func padRight(t theme, value string, width int) string {
	valueWidth := t.Width(value)
	if valueWidth >= width {
		return value
	}
	return value + strings.Repeat(" ", width-valueWidth)
}

func orderedGroups(children []*command) []string {
	seen := make(map[string]bool)
	var groups []string
	for _, child := range children {
		group := commandGroup(child)
		if seen[group] {
			continue
		}
		seen[group] = true
		groups = append(groups, group)
	}
	return groups
}

func commandGroup(c *command) string {
	if c.group != "" {
		return c.group
	}
	return "Commands"
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
		withGroup("Core", versionCommand()),
		withGroup("Core", statusCommand()),
		withGroup("Core", initCommand()),
		withGroup("Node Roles", masterCommand()),
		withGroup("Node Roles", workerCommand()),
		withGroup("Operations", agentCommand()),
		withGroup("Operations", logsCommand()),
		withGroup("Operations", debugCommand()),
		withGroup("Operations", meshCommand()),
		withGroup("Future Surfaces", networkCommand()),
		withGroup("Future Surfaces", nodeCommand()),
		withGroup("Future Surfaces", pgCommand()),
	)

	return root
}

func withGroup(group string, command *command) *command {
	command.group = group
	return command
}
