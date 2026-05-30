package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/tailedbox/tailedbox/internal/status"
)

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeKeyValues(w io.Writer, title string, values [][2]string) {
	fmt.Fprintln(w, title)
	for _, value := range values {
		fmt.Fprintf(w, "  %-32s %s\n", value[0]+":", value[1])
	}
}

func writeMasterStatus(w io.Writer, value status.MasterStatus) {
	fmt.Fprintln(w, "Master Status")
	fmt.Fprintln(w)
	writeKeyValues(w, "Current node", [][2]string{
		{"Node ID", value.Current.NodeID},
		{"Role", value.Current.Role},
		{"Reachability", value.Current.Reachability},
		{"Mesh", string(value.Current.MeshState)},
		{"Last seen", value.Current.LastSeen.Format("2006-01-02 15:04:05 MST")},
		{"Health", string(value.Current.Health)},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Known cluster nodes")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NODE ID\tROLE\tREACHABILITY\tMESH\tLAST SEEN\tHEALTH")
	for _, node := range value.KnownNodes {
		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			node.NodeID,
			node.Role,
			node.Reachability,
			node.MeshState,
			node.LastSeen.Format("2006-01-02 15:04:05 MST"),
			node.Health,
		)
	}
	_ = tw.Flush()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Summary: %d master(s), %d worker(s), %d healthy, %d degraded\n", value.Summary.Masters, value.Summary.Workers, value.Summary.Healthy, value.Summary.Degraded)
}

func writeWorkerStatus(w io.Writer, value status.WorkerStatus) {
	writeKeyValues(w, "Worker Status", [][2]string{
		{"Node ID", value.NodeID},
		{"Role", value.Role},
		{"Initialized", boolString(value.Initialized)},
		{"Joined to master cluster", boolString(value.JoinedToMasterCluster)},
		{"Connected to master cluster", boolString(value.ConnectedToMasterCluster)},
		{"Authenticated", boolString(value.Authenticated)},
		{"Mesh reachable", boolString(value.MeshReachable)},
		{"Mesh", string(value.MeshState)},
		{"Health", string(value.Health)},
	})
}

func writeLocalStatus(w io.Writer, value status.LocalStatus) {
	writeKeyValues(w, "Tailedbox Status", [][2]string{
		{"Node ID", value.NodeID},
		{"Role", value.Role},
		{"Initialized", boolString(value.Initialized)},
		{"Config file", value.ConfigFile},
		{"State directory", value.StateDir},
		{"Log file", value.LogFile},
		{"Health", string(value.Health)},
	})
}

func boolString(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func plannedMessage(area string) string {
	return strings.TrimSpace(area) + " is planned for a later Tailedbox POC part and is intentionally not implemented in Part 1."
}
