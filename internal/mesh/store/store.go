package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	StateDisabled     = "disabled"
	StateStopped      = "stopped"
	StateListening    = "listening"
	StateConnected    = "connected"
	StateDisconnected = "disconnected"
	StateDegraded     = "degraded"

	HealthHealthy  = "healthy"
	HealthDegraded = "degraded"

	SessionStateConnecting   = "connecting"
	SessionStateConnected    = "connected"
	SessionStateDisconnected = "disconnected"
	SessionStateDegraded     = "degraded"
)

type RuntimePaths struct {
	MeshDir    string
	PeersDir   string
	StatusFile string
}

type Status struct {
	Version              int       `json:"version"`
	NodeID               string    `json:"node_id"`
	Role                 string    `json:"role"`
	Enabled              bool      `json:"enabled"`
	State                string    `json:"state"`
	Health               string    `json:"health"`
	ListenUDPPort        int       `json:"listen_udp_port"`
	BoundEndpoint        string    `json:"bound_endpoint,omitempty"`
	StartedAt            time.Time `json:"started_at,omitempty"`
	LastUpdatedAt        time.Time `json:"last_updated_at"`
	PeerCount            int       `json:"peer_count"`
	EstablishedPeerCount int       `json:"established_peer_count"`
	Message              string    `json:"message,omitempty"`
}

type PeerObservation struct {
	Version             int       `json:"version"`
	NodeID              string    `json:"node_id"`
	Role                string    `json:"role"`
	IdentityFingerprint string    `json:"identity_fingerprint"`
	LastEndpoint        string    `json:"last_endpoint,omitempty"`
	LastSeenAt          time.Time `json:"last_seen_at"`
	SessionState        string    `json:"session_state"`
}

func ResolvePaths(paths config.Paths) (RuntimePaths, error) {
	if paths.StateDir == "" {
		return RuntimePaths{}, errors.New("state directory is required for mesh store")
	}
	meshDir := filepath.Join(paths.StateDir, "mesh")
	return RuntimePaths{
		MeshDir:    meshDir,
		PeersDir:   filepath.Join(meshDir, "peers"),
		StatusFile: filepath.Join(meshDir, "status.json"),
	}, nil
}

func EnsureDirs(paths config.Paths) (RuntimePaths, error) {
	runtimePaths, err := ResolvePaths(paths)
	if err != nil {
		return RuntimePaths{}, err
	}
	for _, dir := range []string{runtimePaths.MeshDir, runtimePaths.PeersDir} {
		if err := secrets.EnsurePrivateDir(dir); err != nil {
			return RuntimePaths{}, err
		}
	}
	return runtimePaths, nil
}

func WriteStatus(paths config.Paths, status Status) (bool, error) {
	runtimePaths, err := EnsureDirs(paths)
	if err != nil {
		return false, err
	}
	if status.Version == 0 {
		status.Version = 1
	}
	return secrets.WriteJSONAtomic(runtimePaths.StatusFile, status)
}

func ReadStatus(paths config.Paths) (Status, error) {
	runtimePaths, err := ResolvePaths(paths)
	if err != nil {
		return Status{}, err
	}
	var status Status
	if err := secrets.ReadJSON(runtimePaths.StatusFile, &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func WritePeer(paths config.Paths, peer PeerObservation) (bool, error) {
	if err := validateNodeID(peer.NodeID); err != nil {
		return false, err
	}
	runtimePaths, err := EnsureDirs(paths)
	if err != nil {
		return false, err
	}
	if peer.Version == 0 {
		peer.Version = 1
	}
	return secrets.WriteJSONAtomic(peerPath(runtimePaths, peer.NodeID), peer)
}

func ReadPeer(paths config.Paths, nodeID string) (PeerObservation, error) {
	if err := validateNodeID(nodeID); err != nil {
		return PeerObservation{}, err
	}
	runtimePaths, err := ResolvePaths(paths)
	if err != nil {
		return PeerObservation{}, err
	}
	var peer PeerObservation
	if err := secrets.ReadJSON(peerPath(runtimePaths, nodeID), &peer); err != nil {
		return PeerObservation{}, err
	}
	return peer, nil
}

func ListPeers(paths config.Paths) ([]PeerObservation, error) {
	runtimePaths, err := ResolvePaths(paths)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(runtimePaths.PeersDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read mesh peers directory: %w", err)
	}
	peers := make([]PeerObservation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var peer PeerObservation
		if err := secrets.ReadJSON(filepath.Join(runtimePaths.PeersDir, entry.Name()), &peer); err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].NodeID < peers[j].NodeID
	})
	return peers, nil
}

func peerPath(paths RuntimePaths, nodeID string) string {
	return filepath.Join(paths.PeersDir, nodeID+".json")
}

func validateNodeID(nodeID string) error {
	if nodeID == "" {
		return errors.New("mesh peer node id is required")
	}
	if nodeID == "." || nodeID == ".." || strings.ContainsAny(nodeID, `/\`) {
		return fmt.Errorf("invalid mesh peer node id %q", nodeID)
	}
	return nil
}
