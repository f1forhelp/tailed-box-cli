package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/tailedbox/secureconn/store"
)

const (
	OperationStatus   = "mesh.status"
	OperationPeers    = "mesh.peers"
	OperationPing     = "mesh.ping"
	OperationDiagnose = "mesh.diagnose"

	DefaultTimeout = 5 * time.Second

	maxUnixSocketPath = 100
)

type Request struct {
	Version    int    `json:"version"`
	Operation  string `json:"operation"`
	PeerNodeID string `json:"peer_node_id,omitempty"`
}

type Response struct {
	Version  int                     `json:"version"`
	OK       bool                    `json:"ok"`
	Error    string                  `json:"error,omitempty"`
	Status   *store.Status           `json:"status,omitempty"`
	Peers    []store.PeerObservation `json:"peers,omitempty"`
	Ping     *PingResult             `json:"ping,omitempty"`
	Diagnose *DiagnoseResult         `json:"diagnose,omitempty"`
}

type PingResult struct {
	Version               int    `json:"version"`
	PeerNodeID            string `json:"peer_node_id"`
	Success               bool   `json:"success"`
	Message               string `json:"message,omitempty"`
	LatencyMilliseconds   int64  `json:"latency_milliseconds,omitempty"`
	AgentControlReachable bool   `json:"agent_control_reachable"`
}

type DiagnoseResult struct {
	Version               int      `json:"version"`
	AgentControlReachable bool     `json:"agent_control_reachable"`
	MeshEnabled           bool     `json:"mesh_enabled"`
	UDPTransportReady     bool     `json:"udp_transport_ready"`
	NodeID                string   `json:"node_id,omitempty"`
	Role                  string   `json:"role,omitempty"`
	State                 string   `json:"state,omitempty"`
	Health                string   `json:"health,omitempty"`
	ListenUDPPort         int      `json:"listen_udp_port"`
	BoundEndpoint         string   `json:"bound_endpoint,omitempty"`
	PeerCount             int      `json:"peer_count"`
	EstablishedPeerCount  int      `json:"established_peer_count"`
	ControlSocket         string   `json:"control_socket"`
	StatusFile            string   `json:"status_file"`
	Messages              []string `json:"messages"`
}

type Handler func(context.Context, Request) Response

func SocketPath(paths store.Paths) string {
	direct := filepath.Join(paths.AgentDir, "control.sock")
	if len(direct) <= maxUnixSocketPath {
		return direct
	}
	sum := sha256.Sum256([]byte(filepath.Clean(paths.StateDir)))
	return filepath.Join(os.TempDir(), "tailedbox-agent-"+hex.EncodeToString(sum[:8]), "control.sock")
}

func Listen(paths store.Paths) (net.Listener, string, error) {
	socketPath := SocketPath(paths)
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		return nil, "", fmt.Errorf("create agent control directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(socketPath), 0o700); err != nil {
		return nil, "", fmt.Errorf("secure agent control directory: %w", err)
	}
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, "", fmt.Errorf("agent control path %q exists and is not a socket", socketPath)
		}
		if err := os.Remove(socketPath); err != nil {
			return nil, "", fmt.Errorf("remove stale agent control socket: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, "", fmt.Errorf("stat agent control socket: %w", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, "", fmt.Errorf("listen on agent control socket: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = listener.Close()
		return nil, "", fmt.Errorf("secure agent control socket: %w", err)
	}
	return listener, socketPath, nil
}

func Serve(ctx context.Context, listener net.Listener, handler Handler) {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go handleConn(ctx, conn, handler)
	}
}

func RoundTrip(ctx context.Context, paths store.Paths, request Request) (Response, error) {
	if request.Version == 0 {
		request.Version = 1
	}
	if request.Operation == "" {
		return Response{}, errors.New("mesh control operation is required")
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", SocketPath(paths))
	if err != nil {
		return Response{}, fmt.Errorf("connect to local agent control socket: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return Response{}, fmt.Errorf("write local agent control request: %w", err)
	}
	var response Response
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return Response{}, fmt.Errorf("read local agent control response: %w", err)
	}
	if response.Version == 0 {
		response.Version = 1
	}
	return response, nil
}

func ErrorResponse(err error) Response {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return Response{Version: 1, OK: false, Error: message}
}

func OKResponse() Response {
	return Response{Version: 1, OK: true}
}

func handleConn(ctx context.Context, conn net.Conn, handler Handler) {
	defer conn.Close()
	var request Request
	if err := json.NewDecoder(conn).Decode(&request); err != nil {
		_ = json.NewEncoder(conn).Encode(ErrorResponse(fmt.Errorf("parse local agent control request: %w", err)))
		return
	}
	if request.Version != 0 && request.Version != 1 {
		_ = json.NewEncoder(conn).Encode(ErrorResponse(fmt.Errorf("unsupported local agent control version %d", request.Version)))
		return
	}
	response := handler(ctx, request)
	if response.Version == 0 {
		response.Version = 1
	}
	_ = json.NewEncoder(conn).Encode(response)
}
