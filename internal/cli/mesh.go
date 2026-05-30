package cli

func meshCommand() *command {
	mesh := &command{
		name:        "mesh",
		usage:       "tailedbox mesh <command> [flags]",
		summary:     "Mesh diagnostics and peer commands",
		description: "The secure mesh implementation starts in Parts 6 and 7. Part 1 reserves the CLI namespace.",
	}
	attach(mesh,
		plannedLeaf("status", "tailedbox mesh status [--json]", "Show mesh status", "Mesh status"),
		plannedLeaf("peers", "tailedbox mesh peers [--json]", "List mesh peers", "Mesh peer listing"),
		plannedLeaf("ping", "tailedbox mesh ping <node-id>", "Ping a mesh peer", "Mesh ping"),
		plannedLeaf("diagnose", "tailedbox mesh diagnose [--json]", "Diagnose mesh connectivity", "Mesh diagnostics"),
	)
	return mesh
}
