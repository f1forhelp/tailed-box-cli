package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tailedbox/secureconn/internal/lab"
	"github.com/tailedbox/secureconn/store"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		_, err := tea.NewProgram(newModel(defaultLabRoot())).Run()
		return err
	}
	switch args[0] {
	case "lab":
		return runLab(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printHelp(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runLab(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}
	switch args[0] {
	case "init":
		return runLabInit(args[1:], stdout)
	case "invite":
		return runLabInvite(args[1:], stdout)
	case "join":
		return runLabJoin(args[1:], stdout)
	case "pair":
		return runLabPair(args[1:], stdout)
	case "run":
		return runLabRun(args[1:], stdout)
	case "ping":
		return runLabPing(args[1:], stdout)
	case "status":
		return runLabStatus(args[1:], stdout)
	case "trust":
		return runLabTrust(args[1:], stdout)
	default:
		return fmt.Errorf("unknown lab command %q", args[0])
	}
}

func runLabInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "lab node state directory")
	role := fs.String("role", lab.RoleWorker, "lab node role: master or worker")
	nodeID := fs.String("node-id", "", "optional node id")
	clusterID := fs.String("cluster-id", lab.DefaultClusterID, "lab cluster id")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	node, created, err := lab.Init(lab.InitOptions{
		StateDir:  *stateDir,
		Role:      *role,
		NodeID:    *nodeID,
		ClusterID: *clusterID,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(stdout, map[string]any{"created": created, "node": node})
	}
	fmt.Fprintf(stdout, "lab node %s (%s) ready at %s\n", node.NodeID, node.Role, *stateDir)
	return nil
}

func runLabPair(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab pair", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	masterStateDir := fs.String("master-state-dir", "", "master lab state directory")
	workerStateDir := fs.String("worker-state-dir", "", "worker lab state directory")
	masterEndpoint := fs.String("master-endpoint", "", "master UDP endpoint")
	masterPublicEndpoint := fs.String("master-public-endpoint", "", "master public UDP endpoint")
	masterVPCEndpoint := fs.String("master-vpc-endpoint", "", "master VPC/private UDP endpoint")
	workerEndpoint := fs.String("worker-endpoint", "", "optional worker UDP endpoint")
	workerPublicEndpoint := fs.String("worker-public-endpoint", "", "worker public UDP endpoint")
	workerVPCEndpoint := fs.String("worker-vpc-endpoint", "", "worker VPC/private UDP endpoint")
	oneWay := fs.Bool("one-way", false, "only trust master from worker")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if err := lab.Pair(lab.PairOptions{
		MasterStateDir:       *masterStateDir,
		WorkerStateDir:       *workerStateDir,
		MasterEndpoint:       *masterEndpoint,
		MasterPublicEndpoint: *masterPublicEndpoint,
		MasterVPCEndpoint:    *masterVPCEndpoint,
		WorkerEndpoint:       *workerEndpoint,
		WorkerPublicEndpoint: *workerPublicEndpoint,
		WorkerVPCEndpoint:    *workerVPCEndpoint,
		TrustBothWays:        !*oneWay,
		Now:                  time.Now().UTC(),
	}); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "paired master %s and worker %s\n", *masterStateDir, *workerStateDir)
	return nil
}

func runLabInvite(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected invite command: create or list")
	}
	switch args[0] {
	case "create":
		return runLabInviteCreate(args[1:], stdout)
	case "list":
		return runLabInviteList(args[1:], stdout)
	default:
		return fmt.Errorf("unknown invite command %q", args[0])
	}
}

func runLabInviteCreate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab invite create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "master lab state directory")
	role := fs.String("role", lab.RoleWorker, "role allowed to join")
	ttl := fs.Duration("ttl", lab.DefaultInviteTTL, "invite lifetime")
	publicEndpoint := fs.String("public-endpoint", "", "master public UDP endpoint")
	vpcEndpoint := fs.String("vpc-endpoint", "", "master VPC/private UDP endpoint")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := lab.CreateInvite(lab.InviteOptions{
		StateDir:       *stateDir,
		Role:           *role,
		TTL:            *ttl,
		PublicEndpoint: *publicEndpoint,
		VPCEndpoint:    *vpcEndpoint,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprintf(stdout, "invite %s ready for %s nodes\n", result.Invite.ID, result.Invite.Role)
	fmt.Fprintf(stdout, "code: %s\n", result.Code)
	fmt.Fprintf(stdout, "expires: %s\n", result.Invite.ExpiresAt.Format(time.RFC3339))
	printEndpointLines(stdout, result.Invite.Endpoints)
	return nil
}

func runLabInviteList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab invite list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "master lab state directory")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	invites, err := lab.ListInvites(*stateDir)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(stdout, invites)
	}
	if len(invites) == 0 {
		fmt.Fprintln(stdout, "no invites")
		return nil
	}
	for _, invite := range invites {
		state := "active"
		switch {
		case !invite.UsedAt.IsZero():
			state = "used"
		case time.Now().UTC().After(invite.ExpiresAt):
			state = "expired"
		}
		fmt.Fprintf(stdout, "%s role=%s state=%s expires=%s", invite.ID, invite.Role, state, invite.ExpiresAt.Format(time.RFC3339))
		if invite.Endpoints.Public != "" {
			fmt.Fprintf(stdout, " public=%s", invite.Endpoints.Public)
		}
		if invite.Endpoints.VPC != "" {
			fmt.Fprintf(stdout, " vpc=%s", invite.Endpoints.VPC)
		}
		fmt.Fprintln(stdout)
	}
	return nil
}

func runLabJoin(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab join", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "worker lab state directory")
	code := fs.String("code", "", "invite code")
	masterEndpoint := fs.String("master-endpoint", "", "master UDP endpoint to contact")
	publicEndpoint := fs.String("public-endpoint", "", "optional worker public UDP endpoint")
	vpcEndpoint := fs.String("vpc-endpoint", "", "optional worker VPC/private UDP endpoint")
	timeout := fs.Duration("timeout", 8*time.Second, "join timeout")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	result, err := lab.Join(context.Background(), lab.JoinOptions{
		StateDir:       *stateDir,
		Code:           *code,
		MasterEndpoint: *masterEndpoint,
		PublicEndpoint: *publicEndpoint,
		VPCEndpoint:    *vpcEndpoint,
		Timeout:        *timeout,
		Now:            time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprintf(stdout, "joined master %s via %s\n", result.MasterNodeID, result.MasterEndpoint)
	printEndpointLines(stdout, result.Endpoints)
	return nil
}

func runLabTrust(args []string, stdout io.Writer) error {
	if len(args) > 0 && args[0] == "revoke" {
		return runLabTrustRevoke(args[1:], stdout)
	}
	fs := flag.NewFlagSet("secureconn lab trust", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "lab node state directory")
	peerStateDir := fs.String("peer-state-dir", "", "peer lab state directory")
	endpoint := fs.String("endpoint", "", "optional peer UDP endpoint")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	peer, err := lab.LoadNode(*peerStateDir)
	if err != nil {
		return err
	}
	if err := lab.AddTrust(*stateDir, peer, *endpoint, time.Now().UTC()); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "trusted peer %s from %s\n", peer.NodeID, *stateDir)
	return nil
}

func runLabTrustRevoke(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab trust revoke", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "lab node state directory")
	peerNodeID := fs.String("peer", "", "peer node id to revoke")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if err := lab.RemoveTrust(*stateDir, *peerNodeID); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "revoked trust for peer %s from %s\n", *peerNodeID, *stateDir)
	return nil
}

func runLabRun(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "lab node state directory")
	host := fs.String("host", "127.0.0.1", "UDP listen host")
	port := fs.Int("port", 0, "UDP listen port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	transport, node, err := lab.Start(ctx, *stateDir, *host, *port)
	if err != nil {
		return err
	}
	defer transport.Close()
	fmt.Fprintf(stdout, "secureconn lab node %s (%s) listening on %s\n", node.NodeID, node.Role, transport.BoundEndpoint())
	<-ctx.Done()
	_ = transport.Close()
	_ = lab.WriteStoppedStatus(*stateDir, "secureconn lab listener stopped")
	fmt.Fprintln(stdout, "secureconn lab listener stopped")
	return nil
}

func runLabPing(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab ping", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "lab node state directory")
	peerNodeID := fs.String("peer", "", "peer node id")
	endpoint := fs.String("endpoint", "", "peer UDP endpoint")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	started := time.Now()
	latency, err := lab.Ping(context.Background(), *stateDir, *peerNodeID, *endpoint)
	result := map[string]any{
		"success":              err == nil,
		"peer_node_id":         *peerNodeID,
		"endpoint":             *endpoint,
		"latency_milliseconds": latency.Milliseconds(),
		"elapsed_milliseconds": time.Since(started).Milliseconds(),
	}
	if err != nil {
		result["error"] = err.Error()
	}
	if *jsonOutput {
		return writeJSON(stdout, result)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "pong from %s in %s\n", *peerNodeID, latency)
	return nil
}

func runLabStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("secureconn lab status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stateDir := fs.String("state-dir", "", "lab node state directory")
	jsonOutput := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	node, status, peers, trusted, err := lab.Status(*stateDir)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"node":    node,
		"status":  status,
		"peers":   peers,
		"trusted": trusted,
	}
	if *jsonOutput {
		return writeJSON(stdout, payload)
	}
	printStatus(stdout, node, status, peers, trusted)
	return nil
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "secureconn lab tool")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  secureconn                         open Bubble Tea lab menu")
	fmt.Fprintln(w, "  secureconn lab init --role master --state-dir ./lab/master")
	fmt.Fprintln(w, "  secureconn lab init --role worker --state-dir ./lab/worker")
	fmt.Fprintln(w, "  secureconn lab invite create --state-dir ./lab/master --public-endpoint 203.0.113.10:41677 --vpc-endpoint 10.0.1.5:41677")
	fmt.Fprintln(w, "  secureconn lab join --state-dir ./lab/worker --code <code> --master-endpoint 203.0.113.10:41677")
	fmt.Fprintln(w, "  secureconn lab pair --master-state-dir ./lab/master --worker-state-dir ./lab/worker --master-endpoint 127.0.0.1:41677")
	fmt.Fprintln(w, "  secureconn lab trust revoke --state-dir ./lab/master --peer <node-id>")
	fmt.Fprintln(w, "  secureconn lab run --state-dir ./lab/master --host 127.0.0.1 --port 41677")
	fmt.Fprintln(w, "  secureconn lab ping --state-dir ./lab/worker --peer <master-node-id> --endpoint 127.0.0.1:41677")
	fmt.Fprintln(w, "  secureconn lab status --state-dir ./lab/worker")
}

func printStatus(w io.Writer, node lab.Node, status store.Status, peers []store.PeerObservation, trusted []lab.TrustRecord) {
	fmt.Fprintf(w, "Node:    %s (%s)\n", node.NodeID, node.Role)
	fmt.Fprintf(w, "Cluster: %s\n", node.ClusterID)
	fmt.Fprintf(w, "State:   %s / %s\n", status.State, status.Health)
	if status.BoundEndpoint != "" {
		fmt.Fprintf(w, "Listen:  %s\n", status.BoundEndpoint)
	}
	fmt.Fprintf(w, "Trusted peers: %d\n", len(trusted))
	for _, peer := range trusted {
		endpoint := peer.LastEndpoint
		if endpoint == "" {
			endpoint = "unknown"
		}
		fmt.Fprintf(w, "  - %s (%s) endpoint=%s", peer.NodeID, peer.Role, endpoint)
		if peer.Endpoints.Public != "" {
			fmt.Fprintf(w, " public=%s", peer.Endpoints.Public)
		}
		if peer.Endpoints.VPC != "" {
			fmt.Fprintf(w, " vpc=%s", peer.Endpoints.VPC)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "Observed peers: %d\n", len(peers))
	for _, peer := range peers {
		fmt.Fprintf(w, "  - %s (%s) %s endpoint=%s\n", peer.NodeID, peer.Role, peer.SessionState, peer.LastEndpoint)
	}
}

func printEndpointLines(w io.Writer, endpoints lab.EndpointSet) {
	endpoints = endpoints.Normalized()
	if endpoints.Public != "" {
		fmt.Fprintf(w, "public endpoint: %s\n", endpoints.Public)
	}
	if endpoints.VPC != "" {
		fmt.Fprintf(w, "vpc endpoint: %s\n", endpoints.VPC)
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

type model struct {
	root            string
	cursor          int
	messages        []string
	masterTransport interface{ Close() error }
	masterEndpoint  string
	masterNodeID    string
	inviteCode      string
}

func newModel(root string) model {
	return model{root: root, messages: []string{"select an action"}}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.masterTransport != nil {
				_ = m.masterTransport.Close()
			}
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter":
			return m.runAction()
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "secureconn lab\n")
	fmt.Fprintf(&b, "root: %s\n\n", m.root)
	for i, item := range menuItems {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, item)
	}
	fmt.Fprintf(&b, "\n")
	for _, message := range m.messages {
		fmt.Fprintf(&b, "%s\n", message)
	}
	fmt.Fprintf(&b, "\nq quits\n")
	return b.String()
}

func (m model) runAction() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		return m.withResult("init master", func() error {
			node, _, err := lab.Init(lab.InitOptions{StateDir: m.masterDir(), Role: lab.RoleMaster, NodeID: "lab_master", ClusterID: lab.DefaultClusterID, Now: time.Now().UTC()})
			if err == nil {
				m.masterNodeID = node.NodeID
			}
			return err
		}), nil
	case 1:
		return m.withResult("init worker", func() error {
			_, _, err := lab.Init(lab.InitOptions{StateDir: m.workerDir(), Role: lab.RoleWorker, NodeID: "lab_worker", ClusterID: lab.DefaultClusterID, Now: time.Now().UTC()})
			return err
		}), nil
	case 2:
		return m.withResult("create invite", func() error {
			result, err := lab.CreateInvite(lab.InviteOptions{
				StateDir:       m.masterDir(),
				Role:           lab.RoleWorker,
				TTL:            lab.DefaultInviteTTL,
				PublicEndpoint: lab.Endpoint("127.0.0.1", 41677),
				Now:            time.Now().UTC(),
			})
			if err != nil {
				return err
			}
			m.inviteCode = result.Code
			return nil
		}), nil
	case 3:
		if m.masterTransport != nil {
			m.messages = appendMessage(m.messages, "master listener is already running")
			return m, nil
		}
		ctx := context.Background()
		labTransport, node, err := lab.Start(ctx, m.masterDir(), "127.0.0.1", 41677)
		if err != nil {
			m.messages = appendMessage(m.messages, "start listener failed: "+err.Error())
			return m, nil
		}
		m.masterTransport = labTransport
		m.masterEndpoint = labTransport.BoundEndpoint()
		m.masterNodeID = node.NodeID
		m.messages = appendMessage(m.messages, "master listening on "+m.masterEndpoint)
		return m, nil
	case 4:
		if m.inviteCode == "" {
			m.messages = appendMessage(m.messages, "create an invite before joining")
			return m, nil
		}
		if m.masterEndpoint == "" {
			m.masterEndpoint = lab.Endpoint("127.0.0.1", 41677)
		}
		result, err := lab.Join(context.Background(), lab.JoinOptions{
			StateDir:       m.workerDir(),
			Code:           m.inviteCode,
			MasterEndpoint: m.masterEndpoint,
			Timeout:        8 * time.Second,
			Now:            time.Now().UTC(),
		})
		if err != nil {
			m.messages = appendMessage(m.messages, "join failed: "+err.Error())
			return m, nil
		}
		m.messages = appendMessage(m.messages, "joined master "+result.MasterNodeID)
		return m, nil
	case 5:
		if m.masterEndpoint == "" {
			m.masterEndpoint = lab.Endpoint("127.0.0.1", 41677)
		}
		master, err := lab.LoadNode(m.masterDir())
		if err != nil {
			m.messages = appendMessage(m.messages, "load master failed: "+err.Error())
			return m, nil
		}
		latency, err := lab.Ping(context.Background(), m.workerDir(), master.NodeID, m.masterEndpoint)
		if err != nil {
			m.messages = appendMessage(m.messages, "ping failed: "+err.Error())
			return m, nil
		}
		m.messages = appendMessage(m.messages, "pong from "+master.NodeID+" in "+latency.String())
		return m, nil
	case 6:
		m.messages = statusMessages(m.workerDir())
		return m, nil
	case 7:
		if m.masterTransport != nil {
			_ = m.masterTransport.Close()
			_ = lab.WriteStoppedStatus(m.masterDir(), "secureconn lab listener stopped")
			m.masterTransport = nil
			m.messages = appendMessage(m.messages, "master listener stopped")
		}
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) withResult(label string, fn func() error) model {
	if err := fn(); err != nil {
		m.messages = appendMessage(m.messages, label+" failed: "+err.Error())
		return m
	}
	m.messages = appendMessage(m.messages, label+" ok")
	return m
}

func (m model) masterDir() string {
	return filepath.Join(m.root, "master")
}

func (m model) workerDir() string {
	return filepath.Join(m.root, "worker")
}

func statusMessages(stateDir string) []string {
	node, status, peers, trusted, err := lab.Status(stateDir)
	if err != nil {
		return []string{"status failed: " + err.Error()}
	}
	lines := []string{
		"node " + node.NodeID + " (" + node.Role + ")",
		"cluster " + node.ClusterID,
		"state " + status.State + " / " + status.Health,
		"trusted peers " + strconv.Itoa(len(trusted)),
		"observed peers " + strconv.Itoa(len(peers)),
	}
	for _, peer := range peers {
		lines = append(lines, "peer "+peer.NodeID+" "+peer.SessionState+" "+peer.LastEndpoint)
	}
	return lines
}

func appendMessage(messages []string, message string) []string {
	messages = append(messages, message)
	if len(messages) > 8 {
		return messages[len(messages)-8:]
	}
	return messages
}

func defaultLabRoot() string {
	return filepath.Join(".", "lab")
}

var menuItems = []string{
	"init master at ./lab/master",
	"init worker at ./lab/worker",
	"create worker invite for 127.0.0.1:41677",
	"start master listener on 127.0.0.1:41677",
	"join worker with invite",
	"ping master from worker",
	"show worker status",
	"quit",
}
