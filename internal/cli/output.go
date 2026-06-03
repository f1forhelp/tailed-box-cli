package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/tailedbox/tailedbox/internal/agent"
	"github.com/tailedbox/tailedbox/internal/mesh/control"
	"github.com/tailedbox/tailedbox/internal/mesh/store"
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
		{"Identity", readyString(t, value.Current.IdentityReady)},
		{"Agent config", readyString(t, value.Current.AgentConfigReady)},
		{"Identity fingerprint", optionalString(value.Current.IdentityFingerprint, "missing")},
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
			readyString(t, node.IdentityReady),
			readyString(t, node.AgentConfigReady),
			node.LastSeen.Format("2006-01-02 15:04:05 MST"),
			t.Health(string(node.Health)),
		})
	}
	fmt.Fprintln(w, renderTable(t, []string{"NODE ID", "ROLE", "REACHABILITY", "MESH", "IDENTITY", "AGENT", "LAST SEEN", "HEALTH"}, rows))
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
		{"Identity", readyString(t, value.IdentityReady)},
		{"Agent config", readyString(t, value.AgentConfigReady)},
		{"Identity fingerprint", optionalString(value.IdentityFingerprint, "missing")},
		{"Joined to master cluster", t.Bool(value.JoinedToMasterCluster)},
		{"Cluster ID", optionalString(value.ClusterID, "not joined")},
		{"Cluster name", optionalString(value.ClusterName, "not joined")},
		{"Reconnect lease", formatOptionalTime(value.ReconnectLeaseExpiresAt, "not joined")},
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
		{"Identity", readyString(t, value.IdentityReady)},
		{"Agent config", readyString(t, value.AgentConfigReady)},
		{"Identity fingerprint", optionalString(value.IdentityFingerprint, "missing")},
		{"Config file", value.ConfigFile},
		{"State directory", value.StateDir},
		{"Log file", value.LogFile},
		{"Health", t.Health(string(value.Health))},
	})
}

func writeAgentStatus(w io.Writer, t theme, value agent.Status) {
	fmt.Fprintln(w, t.Title("Agent Status"))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Local agent", [][2]string{
		{"Node ID", optionalString(value.NodeID, "unassigned")},
		{"Role", optionalString(value.Role, "uninitialized")},
		{"State", t.Health(value.State)},
		{"Health", t.Health(value.Health)},
		{"Running", t.Bool(value.Running)},
		{"PID", optionalInt(value.PID, "not running")},
		{"Started at", formatOptionalTime(value.StartedAt, "not running")},
		{"Last heartbeat", formatOptionalTime(value.LastHeartbeatAt, "never")},
		{"Heartbeat age", formatDurationSeconds(value.HeartbeatAgeSeconds)},
		{"Uptime", formatDurationSeconds(value.UptimeSeconds)},
		{"Memory alloc", formatBytes(value.MemoryAllocBytes)},
		{"Memory sys", formatBytes(value.MemorySysBytes)},
		{"Goroutines", strconv.Itoa(value.Goroutines)},
		{"Status file", value.AgentStatusFile},
		{"Systemd service", value.SystemdServiceName},
		{"Systemd unit", value.SystemdUnitPath},
	})
	if value.Message != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.NoteLine(value.Message))
	}
}

func writeMeshConfigResult(w io.Writer, t theme, message string, value meshConfigResult) {
	fmt.Fprintln(w, t.SuccessLine(message))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Mesh config", [][2]string{
		{"Enabled", t.Bool(value.Mesh.Enabled)},
		{"Provider", value.Mesh.Provider},
		{"Listen UDP port", optionalInt(value.Mesh.ListenUDPPort, "not configured")},
		{"Changed", t.Bool(value.Changed)},
		{"Agent config", value.AgentConfigFile},
	})
}

func writeMeshStatus(w io.Writer, t theme, value store.Status, live bool) {
	fmt.Fprintln(w, t.Title("Mesh Status"))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Local mesh", [][2]string{
		{"Node ID", optionalString(value.NodeID, "unassigned")},
		{"Role", optionalString(value.Role, "uninitialized")},
		{"Agent control", reachableString(t, live)},
		{"Enabled", t.Bool(value.Enabled)},
		{"State", t.Mesh(value.State)},
		{"Health", t.Health(value.Health)},
		{"Listen UDP port", optionalInt(value.ListenUDPPort, "not configured")},
		{"Bound endpoint", optionalString(value.BoundEndpoint, "not listening")},
		{"Started at", formatOptionalTime(value.StartedAt, "not running")},
		{"Last updated", formatOptionalTime(value.LastUpdatedAt, "never")},
		{"Peers", strconv.Itoa(value.PeerCount)},
		{"Connected peers", strconv.Itoa(value.EstablishedPeerCount)},
	})
	if value.Message != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.NoteLine(value.Message))
	}
}

func writeMeshPeers(w io.Writer, t theme, peers []store.PeerObservation, live bool) {
	fmt.Fprintln(w, t.Title("Mesh Peers"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Section("Source"))
	fmt.Fprintf(w, "  %s  %s\n", t.Label("Agent control:"), reachableString(t, live))
	fmt.Fprintln(w)
	if len(peers) == 0 {
		fmt.Fprintln(w, t.NoteLine("No mesh peer observations recorded yet."))
		return
	}
	rows := make([][]string, 0, len(peers))
	for _, peer := range peers {
		rows = append(rows, []string{
			peer.NodeID,
			peer.Role,
			peer.IdentityFingerprint,
			optionalString(peer.LastEndpoint, "unknown"),
			t.Mesh(peer.SessionState),
			formatOptionalTime(peer.LastSeenAt, "never"),
		})
	}
	fmt.Fprintln(w, renderTable(t, []string{"NODE ID", "ROLE", "FINGERPRINT", "ENDPOINT", "SESSION", "LAST SEEN"}, rows))
}

func writeMeshPing(w io.Writer, t theme, value control.PingResult) {
	if value.Success {
		fmt.Fprintln(w, t.SuccessLine("Mesh ping succeeded."))
	} else {
		fmt.Fprintf(w, "%s Mesh ping did not complete.\n", t.Warning("WARN"))
	}
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Ping", [][2]string{
		{"Peer node", value.PeerNodeID},
		{"Agent control", reachableString(t, value.AgentControlReachable)},
		{"Success", t.Bool(value.Success)},
		{"Latency", formatDurationSeconds(value.LatencyMilliseconds / 1000)},
	})
	if value.Message != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.NoteLine(value.Message))
	}
}

func writeMeshDiagnose(w io.Writer, t theme, value control.DiagnoseResult) {
	fmt.Fprintln(w, t.Title("Mesh Diagnostics"))
	fmt.Fprintln(w)
	writeKeyValues(w, t, "Readiness", [][2]string{
		{"Node ID", optionalString(value.NodeID, "unassigned")},
		{"Role", optionalString(value.Role, "uninitialized")},
		{"Agent control", reachableString(t, value.AgentControlReachable)},
		{"Mesh enabled", t.Bool(value.MeshEnabled)},
		{"UDP transport", readyString(t, value.UDPTransportReady)},
		{"State", t.Mesh(value.State)},
		{"Health", t.Health(value.Health)},
		{"Listen UDP port", optionalInt(value.ListenUDPPort, "not configured")},
		{"Bound endpoint", optionalString(value.BoundEndpoint, "not listening")},
		{"Peers", strconv.Itoa(value.PeerCount)},
		{"Connected peers", strconv.Itoa(value.EstablishedPeerCount)},
		{"Control socket", value.ControlSocket},
		{"Status file", value.StatusFile},
	})
	if len(value.Messages) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, t.Section("Findings"))
		for _, message := range value.Messages {
			fmt.Fprintln(w, "  "+t.NoteLine(message))
		}
	}
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
	return strings.TrimSpace(area) + " is planned for a later Tailedbox POC part and is intentionally not implemented yet."
}

func readyString(t theme, ready bool) string {
	if ready {
		return t.Success("ready")
	}
	return t.Warning("missing")
}

func reachableString(t theme, reachable bool) string {
	if reachable {
		return t.Success("reachable")
	}
	return t.Warning("not reachable")
}

func optionalString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func formatOptionalTime(value time.Time, fallback string) string {
	if value.IsZero() {
		return fallback
	}
	return value.Format("2006-01-02 15:04:05 MST")
}

func optionalInt(value int, fallback string) string {
	if value == 0 {
		return fallback
	}
	return strconv.Itoa(value)
}

func formatDurationSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	return (time.Duration(seconds) * time.Second).String()
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KiB", "MiB", "GiB", "TiB"} {
		value = value / unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f PiB", value/unit)
}
