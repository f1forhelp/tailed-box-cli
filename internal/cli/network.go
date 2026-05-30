package cli

func networkCommand() *command {
	network := &command{
		name:        "network",
		usage:       "tailedbox network <command> [flags]",
		summary:     "Network provider commands",
		description: "Network provider support begins with the built-in tailedbox-mesh provider in later POC parts.",
	}
	attach(network,
		plannedLeaf("create", "tailedbox network create --driver tailedbox-mesh", "Create a Tailedbox network", "Network provider creation"),
		plannedLeaf("status", "tailedbox network status [--json]", "Show network status", "Network status"),
		plannedLeaf("peers", "tailedbox network peers [--json]", "List network peers", "Network peer listing"),
	)
	return network
}
