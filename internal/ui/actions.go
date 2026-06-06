package ui

import "strings"

type Action struct {
	Title         string
	Description   string
	Args          []string
	Inputs        []ActionInput
	Kind          ActionKind
	DefaultCancel bool
}

type ActionKind string

const (
	ActionKindQuick  ActionKind = "quick"
	ActionKindStream ActionKind = "stream"
)

type ActionInput struct {
	Label       string
	Description string
	Flag        string
	Default     string
	Required    bool
	Secret      bool
}

func (a Action) HasInputs() bool {
	return len(a.Inputs) > 0
}

func (a Action) EffectiveKind() ActionKind {
	if a.Kind == "" {
		return ActionKindQuick
	}
	return a.Kind
}

func (a Action) BuildArgs(values []string) []string {
	return a.argsFromInputs(values, false)
}

func (a Action) PreviewArgs(values []string) []string {
	return a.argsFromInputs(values, true)
}

func (a Action) argsFromInputs(values []string, preview bool) []string {
	args := append([]string(nil), a.Args...)
	for i, input := range a.Inputs {
		value := inputValue(values, i, input.Default)
		if value == "" && preview && input.Required {
			value = "<" + strings.ToLower(strings.ReplaceAll(input.Label, " ", "-")) + ">"
		}
		if value == "" {
			continue
		}
		if preview && input.Secret {
			value = "[hidden]"
		}
		if input.Flag != "" {
			args = append(args, "--"+input.Flag, value)
			continue
		}
		args = append(args, value)
	}
	return args
}

func (a Action) ValidateInputs(values []string) (int, string) {
	for i, input := range a.Inputs {
		if input.Required && inputValue(values, i, input.Default) == "" {
			return i, input.Label + " is required."
		}
	}
	return -1, ""
}

func inputValue(values []string, index int, fallback string) string {
	value := ""
	if index < len(values) {
		value = strings.TrimSpace(values[index])
	}
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	return value
}

func DefaultActions() []Action {
	return cloneActions([]Action{
		{
			Title:       "Core: status",
			Description: "Show role-aware local status for this node.",
			Args:        []string{"status"},
		},
		{
			Title:       "Core: version",
			Description: "Show build and Go runtime version information.",
			Args:        []string{"version"},
		},
		{
			Title:       "Core: initialize as master",
			Description: "Create master role metadata, identity, and local state.",
			Args:        []string{"init", "--role", "master"},
		},
		{
			Title:       "Core: initialize as worker",
			Description: "Create worker role metadata, identity, and local state.",
			Args:        []string{"init", "--role", "worker"},
		},
		{
			Title:       "Master: initialize",
			Description: "Initialize this server as a master using the master namespace alias.",
			Args:        []string{"master", "init"},
		},
		{
			Title:       "Master: status",
			Description: "Show the current master and trusted cluster nodes.",
			Args:        []string{"master", "status"},
		},
		{
			Title:       "Master: join cluster",
			Description: "Join this master to an existing cluster with a one-time code.",
			Args:        []string{"master", "join"},
			Inputs: []ActionInput{
				{
					Label:       "Join code",
					Description: "Paste the one-time master join code printed by the issuing master.",
					Flag:        "code",
					Required:    true,
					Secret:      true,
				},
				{
					Label:       "Master state dir",
					Description: "Path to the issuing master's state directory.",
					Flag:        "master-state-dir",
					Required:    true,
				},
			},
		},
		{
			Title:       "Master: create worker join code",
			Description: "Issue a one-time 15 minute enrollment code for a worker.",
			Args:        []string{"master", "join-code", "create", "--role", "worker", "--ttl", "15m"},
		},
		{
			Title:       "Master: create master join code",
			Description: "Issue a one-time 15 minute enrollment code for another master.",
			Args:        []string{"master", "join-code", "create", "--role", "master", "--ttl", "15m"},
		},
		{
			Title:       "Worker: initialize",
			Description: "Initialize this server as a worker using the worker namespace alias.",
			Args:        []string{"worker", "init"},
		},
		{
			Title:       "Worker: status",
			Description: "Show this worker's local join and mesh readiness.",
			Args:        []string{"worker", "status"},
		},
		{
			Title:       "Worker: join cluster",
			Description: "Join this worker to a master cluster with a one-time code.",
			Args:        []string{"worker", "join"},
			Inputs: []ActionInput{
				{
					Label:       "Join code",
					Description: "Paste the one-time worker join code printed by the master.",
					Flag:        "code",
					Required:    true,
					Secret:      true,
				},
				{
					Label:       "Master state dir",
					Description: "Path to the issuing master's state directory.",
					Flag:        "master-state-dir",
					Required:    true,
				},
			},
		},
		{
			Title:       "Agent: run foreground",
			Description: "Run the local agent in the foreground until stopped.",
			Args:        []string{"agent", "run"},
			Kind:        ActionKindStream,
		},
		{
			Title:       "Agent: status",
			Description: "Show local agent heartbeat, uptime, and memory usage.",
			Args:        []string{"agent", "status"},
		},
		{
			Title:       "Agent: install dry run",
			Description: "Preview the systemd unit without writing files or starting services.",
			Args:        []string{"agent", "install", "--dry-run"},
		},
		{
			Title:       "Agent: install and start",
			Description: "Install the systemd agent service and start it.",
			Args:        []string{"agent", "install", "--start"},
		},
		{
			Title:       "Agent: uninstall dry run",
			Description: "Preview removal of the systemd unit without changing systemd.",
			Args:        []string{"agent", "uninstall", "--dry-run"},
		},
		{
			Title:       "Agent: uninstall service",
			Description: "Disable, stop, and remove the systemd agent service.",
			Args:        []string{"agent", "uninstall"},
		},
		{
			Title:       "Agent: start",
			Description: "Start the systemd agent service.",
			Args:        []string{"agent", "start"},
		},
		{
			Title:       "Agent: stop",
			Description: "Stop the systemd agent service.",
			Args:        []string{"agent", "stop"},
		},
		{
			Title:       "Agent: restart",
			Description: "Restart the systemd agent service.",
			Args:        []string{"agent", "restart"},
		},
		{
			Title:       "Agent: logs",
			Description: "Show recent local agent logs.",
			Args:        []string{"agent", "logs", "--lines", "100"},
		},
		{
			Title:       "Logs: recent",
			Description: "Show recent local Tailedbox JSONL logs.",
			Args:        []string{"logs", "--lines", "100"},
		},
		{
			Title:       "Logs: follow",
			Description: "Show recent logs and keep following new entries.",
			Args:        []string{"logs", "--follow", "--lines", "100"},
			Kind:        ActionKindStream,
		},
		{
			Title:       "Debug: logs enable",
			Description: "Enable opt-in deep debug logs while preserving redaction.",
			Args:        []string{"debug", "logs", "enable"},
		},
		{
			Title:       "Debug: logs disable",
			Description: "Disable deep debug logs.",
			Args:        []string{"debug", "logs", "disable"},
		},
		{
			Title:       "Mesh: enable",
			Description: "Enable the mesh runtime in the local agent config.",
			Args:        []string{"mesh", "enable"},
		},
		{
			Title:       "Mesh: enable default master port",
			Description: "Enable the mesh runtime with the default UDP listener port.",
			Args:        []string{"mesh", "enable", "--listen-udp-port", "41677"},
		},
		{
			Title:       "Mesh: enable custom port",
			Description: "Enable the mesh runtime with a specific UDP listener port.",
			Args:        []string{"mesh", "enable"},
			Inputs: []ActionInput{
				{
					Label:       "UDP port",
					Description: "UDP port for mesh listeners.",
					Flag:        "listen-udp-port",
					Default:     "41677",
					Required:    true,
				},
			},
		},
		{
			Title:       "Mesh: set master endpoint",
			Description: "Enable mesh and store the master UDP endpoint for this worker.",
			Args:        []string{"mesh", "enable"},
			Inputs: []ActionInput{
				{
					Label:       "Master endpoint",
					Description: "Reachable master endpoint in host:port form.",
					Flag:        "master-endpoint",
					Required:    true,
				},
			},
		},
		{
			Title:       "Mesh: disable",
			Description: "Disable the mesh runtime without changing identity or enrollment state.",
			Args:        []string{"mesh", "disable"},
		},
		{
			Title:       "Mesh: status",
			Description: "Show local mesh runtime status from the agent or state files.",
			Args:        []string{"mesh", "status"},
		},
		{
			Title:       "Mesh: peers",
			Description: "List observed mesh peers from the agent or state files.",
			Args:        []string{"mesh", "peers"},
		},
		{
			Title:       "Mesh: ping peer",
			Description: "Ping a mesh peer through the running local agent.",
			Args:        []string{"mesh", "ping"},
			Inputs: []ActionInput{
				{
					Label:       "Peer node ID",
					Description: "Node ID of the peer to ping.",
					Required:    true,
				},
			},
		},
		{
			Title:       "Mesh: diagnose",
			Description: "Diagnose local mesh readiness and agent control reachability.",
			Args:        []string{"mesh", "diagnose"},
		},
		{
			Title:       "Network: create",
			Description: "Open the planned built-in mesh network creation surface.",
			Args:        []string{"network", "create", "--driver", "tailedbox-mesh"},
		},
		{
			Title:       "Network: status",
			Description: "Open the planned network status surface.",
			Args:        []string{"network", "status"},
		},
		{
			Title:       "Network: peers",
			Description: "Open the planned network peer listing surface.",
			Args:        []string{"network", "peers"},
		},
		{
			Title:       "Node: list",
			Description: "Open the planned cluster node inventory surface.",
			Args:        []string{"node", "list"},
		},
		{
			Title:       "Node: approve",
			Description: "Approve a pending node by node ID when this planned surface is implemented.",
			Args:        []string{"node", "approve"},
			Inputs: []ActionInput{
				{
					Label:       "Node ID",
					Description: "Pending node ID to approve.",
					Required:    true,
				},
			},
		},
		{
			Title:       "PostgreSQL: init",
			Description: "Open the planned PostgreSQL metadata preparation surface.",
			Args:        []string{"pg", "init"},
		},
		{
			Title:       "PostgreSQL: deploy",
			Description: "Open the planned PostgreSQL deployment surface.",
			Args:        []string{"pg", "deploy"},
		},
		{
			Title:       "PostgreSQL: status",
			Description: "Open the planned PostgreSQL status surface.",
			Args:        []string{"pg", "status"},
		},
		{
			Title:       "PostgreSQL: failover",
			Description: "Open the planned PostgreSQL failover surface.",
			Args:        []string{"pg", "failover"},
		},
		{
			Title:       "PostgreSQL: backup",
			Description: "Open the planned PostgreSQL backup surface.",
			Args:        []string{"pg", "backup"},
		},
		{
			Title:       "PostgreSQL: restore",
			Description: "Open the planned PostgreSQL restore surface.",
			Args:        []string{"pg", "restore"},
		},
		{
			Title:       "System: uninstall dry run",
			Description: "Preview removal of local Tailedbox config, state, logs, sockets, identities, and trust files.",
			Args:        []string{"uninstall", "--dry-run"},
		},
		{
			Title:       "System: uninstall all dry run",
			Description: "Preview identity/state cleanup plus systemd service removal.",
			Args:        []string{"uninstall", "--dry-run", "--systemd"},
		},
		{
			Title:         "System: uninstall local files",
			Description:   "Delete local Tailedbox identity, state, config, logs, and trust files after exact confirmation.",
			Args:          []string{"uninstall"},
			DefaultCancel: true,
			Inputs: []ActionInput{
				{
					Label:       "Confirm delete",
					Description: "Type DELETE to remove local Tailedbox files.",
					Flag:        "confirm-delete",
					Required:    true,
				},
			},
		},
		{
			Title:         "System: uninstall service and local files",
			Description:   "Disable the systemd service and delete local identity, state, config, logs, and trust files.",
			Args:          []string{"uninstall", "--systemd"},
			DefaultCancel: true,
			Inputs: []ActionInput{
				{
					Label:       "Confirm delete",
					Description: "Type DELETE to remove the service and local Tailedbox files.",
					Flag:        "confirm-delete",
					Required:    true,
				},
			},
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
		cloned[i].Inputs = append([]ActionInput(nil), action.Inputs...)
	}
	return cloned
}
