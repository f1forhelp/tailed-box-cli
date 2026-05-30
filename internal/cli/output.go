package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/tailedbox/tailedbox/internal/status"
)

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeKeyValues(w io.Writer, t theme, title string, values [][2]string) {
	fmt.Fprintln(w, t.Section(title))
	width := 0
	for _, value := range values {
		label := value[0] + ":"
		if labelWidth := t.Width(label); labelWidth > width {
			width = labelWidth
		}
	}
	for _, value := range values {
		label := padRight(t, value[0]+":", width)
		fmt.Fprintf(w, "  %s  %s\n", t.Label(label), value[1])
	}
}

func writeMasterStatus(w io.Writer, t theme, value status.MasterStatus) {
	fmt.Fprintln(w, t.Title("Master Status"))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Current node", [][2]string{
		{"Node ID", value.Current.NodeID},
		{"Role", value.Current.Role},
		{"Reachability", value.Current.Reachability},
		{"Mesh", t.Mesh(string(value.Current.MeshState))},
		{"Last seen", value.Current.LastSeen.Format("2006-01-02 15:04:05 MST")},
		{"Health", t.Health(string(value.Current.Health))},
	})
	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Section("Known cluster nodes"))
	rows := make([][]string, 0, len(value.KnownNodes))
	for _, node := range value.KnownNodes {
		rows = append(rows, []string{
			node.NodeID,
			node.Role,
			node.Reachability,
			t.Mesh(string(node.MeshState)),
			node.LastSeen.Format("2006-01-02 15:04:05 MST"),
			t.Health(string(node.Health)),
		})
	}
	fmt.Fprintln(w, renderTable(t, []string{"NODE ID", "ROLE", "REACHABILITY", "MESH", "LAST SEEN", "HEALTH"}, rows))
	fmt.Fprintln(w)
	fmt.Fprintf(
		w,
		"%s %d master(s), %d worker(s), %s healthy, %s degraded\n",
		t.Section("Summary:"),
		value.Summary.Masters,
		value.Summary.Workers,
		t.Success(fmt.Sprint(value.Summary.Healthy)),
		t.Warning(fmt.Sprint(value.Summary.Degraded)),
	)
}

func writeWorkerStatus(w io.Writer, t theme, value status.WorkerStatus) {
	fmt.Fprintln(w, t.Title("Worker Status"))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Local node", [][2]string{
		{"Node ID", value.NodeID},
		{"Role", value.Role},
		{"Initialized", t.Bool(value.Initialized)},
		{"Joined to master cluster", t.Bool(value.JoinedToMasterCluster)},
		{"Connected to master cluster", t.Bool(value.ConnectedToMasterCluster)},
		{"Authenticated", t.Bool(value.Authenticated)},
		{"Mesh reachable", t.Bool(value.MeshReachable)},
		{"Mesh", t.Mesh(string(value.MeshState))},
		{"Health", t.Health(string(value.Health))},
	})
}

func writeLocalStatus(w io.Writer, t theme, value status.LocalStatus) {
	fmt.Fprintln(w, t.Title("Tailedbox Status"))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Local node", [][2]string{
		{"Node ID", value.NodeID},
		{"Role", value.Role},
		{"Initialized", t.Bool(value.Initialized)},
		{"Config file", value.ConfigFile},
		{"State directory", value.StateDir},
		{"Log file", value.LogFile},
		{"Health", t.Health(string(value.Health))},
	})
}

func renderTable(t theme, headers []string, rows [][]string) string {
	return table.New().
		Border(lipgloss.ASCIIBorder()).
		BorderStyle(t.TableBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		BorderHeader(true).
		BorderColumn(true).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return t.TableHeader().Padding(0, 1)
			}
			return t.TableCell().Padding(0, 1)
		}).
		Render()
}

func plannedMessage(area string) string {
	return strings.TrimSpace(area) + " is planned for a later Tailedbox POC part and is intentionally not implemented in Part 1."
}
