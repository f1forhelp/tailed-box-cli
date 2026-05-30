package cli

func nodeCommand() *command {
	node := &command{
		name:        "node",
		usage:       "tailedbox node <command> [flags]",
		summary:     "Cluster node inventory commands",
		description: "Node inventory and approvals become meaningful after enrollment and mesh state exist.",
	}
	attach(node,
		plannedLeaf("list", "tailedbox node list [--json]", "List known cluster nodes", "Node inventory"),
		plannedLeaf("approve", "tailedbox node approve <node-id>", "Approve a pending node", "Node approval"),
	)
	return node
}
