package ui

type Action struct {
	Title       string
	Description string
	Args        []string
}

func DefaultActions() []Action {
	return cloneActions([]Action{
		{
			Title:       "Status",
			Description: "Show role-aware local status for this node.",
			Args:        []string{"status"},
		},
		{
			Title:       "Agent status",
			Description: "Show local agent heartbeat, uptime, and memory usage.",
			Args:        []string{"agent", "status"},
		},
		{
			Title:       "Initialize as master",
			Description: "Create master role metadata, identity, and local state.",
			Args:        []string{"init", "--role", "master"},
		},
		{
			Title:       "Initialize as worker",
			Description: "Create worker role metadata, identity, and local state.",
			Args:        []string{"init", "--role", "worker"},
		},
		{
			Title:       "Master status",
			Description: "Show the current master and trusted cluster nodes.",
			Args:        []string{"master", "status"},
		},
		{
			Title:       "Worker status",
			Description: "Show this worker's local join and mesh readiness.",
			Args:        []string{"worker", "status"},
		},
		{
			Title:       "Create worker join code",
			Description: "Issue a one-time 15 minute enrollment code for a worker.",
			Args:        []string{"master", "join-code", "create", "--role", "worker", "--ttl", "15m"},
		},
		{
			Title:       "Create master join code",
			Description: "Issue a one-time 15 minute enrollment code for another master.",
			Args:        []string{"master", "join-code", "create", "--role", "master", "--ttl", "15m"},
		},
		{
			Title:       "Recent logs",
			Description: "Show recent local Tailedbox JSONL logs.",
			Args:        []string{"logs", "--lines", "50"},
		},
		{
			Title:       "Version",
			Description: "Show build and Go runtime version information.",
			Args:        []string{"version"},
		},
		{
			Title:       "Help",
			Description: "Print the full command reference.",
			Args:        []string{"--help"},
		},
		{
			Title:       "Exit",
			Description: "Close the Tailedbox menu.",
			Args:        nil,
		},
	})
}

func cloneActions(actions []Action) []Action {
	cloned := make([]Action, len(actions))
	for i, action := range actions {
		cloned[i] = action
		cloned[i].Args = append([]string(nil), action.Args...)
	}
	return cloned
}
