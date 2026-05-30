package cli

func pgCommand() *command {
	pg := &command{
		name:        "pg",
		usage:       "tailedbox pg <command> [flags]",
		summary:     "PostgreSQL service commands",
		description: "PostgreSQL is intentionally reserved for a future managed-service module after the secure mesh POC.",
	}
	attach(pg,
		plannedLeaf("init", "tailedbox pg init", "Prepare PostgreSQL service metadata", "PostgreSQL service module"),
		plannedLeaf("deploy", "tailedbox pg deploy [--runtime docker|native|nixos]", "Deploy PostgreSQL", "PostgreSQL service module"),
		plannedLeaf("status", "tailedbox pg status [--json]", "Show PostgreSQL status", "PostgreSQL service module"),
		plannedLeaf("failover", "tailedbox pg failover", "Fail over PostgreSQL safely", "PostgreSQL service module"),
		plannedLeaf("backup", "tailedbox pg backup <command>", "Manage PostgreSQL backups", "PostgreSQL service module"),
		plannedLeaf("restore", "tailedbox pg restore", "Restore PostgreSQL from backup", "PostgreSQL service module"),
	)
	return pg
}
