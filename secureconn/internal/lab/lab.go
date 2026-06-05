package lab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tailedbox/secureconn/identity"
	"github.com/tailedbox/secureconn/internal/private"
	"github.com/tailedbox/secureconn/store"
	"github.com/tailedbox/secureconn/transport"
)

const (
	DefaultClusterID = "secureconn-lab"
	RoleMaster       = "master"
	RoleWorker       = "worker"
)

type Node struct {
	Version         int                     `json:"version"`
	NodeID          string                  `json:"node_id"`
	Role            string                  `json:"role"`
	ClusterID       string                  `json:"cluster_id"`
	PublicIdentity  identity.PublicIdentity `json:"public_identity"`
	PrivateIdentity string                  `json:"private_identity"`
	CreatedAt       time.Time               `json:"created_at"`
}

type TrustRecord struct {
	Version             int                     `json:"version"`
	NodeID              string                  `json:"node_id"`
	Role                string                  `json:"role"`
	ClusterID           string                  `json:"cluster_id"`
	IdentityFingerprint string                  `json:"identity_fingerprint"`
	PublicIdentity      identity.PublicIdentity `json:"public_identity"`
	LastEndpoint        string                  `json:"last_endpoint,omitempty"`
	Endpoints           EndpointSet             `json:"endpoints,omitempty"`
	TrustedAt           time.Time               `json:"trusted_at"`
}

type EndpointSet struct {
	Public string `json:"public,omitempty"`
	VPC    string `json:"vpc,omitempty"`
	Last   string `json:"last,omitempty"`
}

type TrustStore struct {
	Version int                    `json:"version"`
	Peers   map[string]TrustRecord `json:"peers"`
}

type InitOptions struct {
	StateDir  string
	Role      string
	NodeID    string
	ClusterID string
	Now       time.Time
}

type PairOptions struct {
	MasterStateDir       string
	WorkerStateDir       string
	MasterEndpoint       string
	MasterPublicEndpoint string
	MasterVPCEndpoint    string
	WorkerEndpoint       string
	WorkerPublicEndpoint string
	WorkerVPCEndpoint    string
	TrustBothWays        bool
	Now                  time.Time
}

func Init(opts InitOptions) (Node, bool, error) {
	if opts.StateDir == "" {
		return Node{}, false, errors.New("state directory is required")
	}
	role := strings.TrimSpace(opts.Role)
	if role != RoleMaster && role != RoleWorker {
		return Node{}, false, fmt.Errorf("role must be %q or %q", RoleMaster, RoleWorker)
	}
	if existing, err := LoadNode(opts.StateDir); err == nil {
		if existing.Role != role {
			return Node{}, false, fmt.Errorf("lab node is already initialized as %s", existing.Role)
		}
		return existing, false, nil
	} else if !errors.Is(unwrapPathError(err), os.ErrNotExist) {
		return Node{}, false, err
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nodeID := strings.TrimSpace(opts.NodeID)
	if nodeID == "" {
		nodeID = "lab_" + role + "_" + strings.ToLower(randomID(6))
	}
	clusterID := strings.TrimSpace(opts.ClusterID)
	if clusterID == "" {
		clusterID = DefaultClusterID
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Node{}, false, fmt.Errorf("generate lab node identity: %w", err)
	}
	publicIdentity := identity.PublicIdentity{
		Version:              1,
		NodeID:               nodeID,
		Algorithm:            identity.AlgorithmEd25519,
		PublicKey:            base64.RawStdEncoding.EncodeToString(publicKey),
		PublicKeyFingerprint: identity.Fingerprint(publicKey),
		CreatedAt:            now,
	}
	node := Node{
		Version:         1,
		NodeID:          nodeID,
		Role:            role,
		ClusterID:       clusterID,
		PublicIdentity:  publicIdentity,
		PrivateIdentity: base64.RawStdEncoding.EncodeToString(privateKey),
		CreatedAt:       now,
	}
	if _, err := private.WriteJSONAtomic(nodeFile(opts.StateDir), node); err != nil {
		return Node{}, false, err
	}
	trust := TrustStore{Version: 1, Peers: map[string]TrustRecord{}}
	if _, err := private.WriteJSONAtomic(trustFile(opts.StateDir), trust); err != nil {
		return Node{}, false, err
	}
	return node, true, nil
}

func LoadNode(stateDir string) (Node, error) {
	var node Node
	if err := private.ReadJSON(nodeFile(stateDir), &node); err != nil {
		return Node{}, err
	}
	if node.Version != 1 {
		return Node{}, fmt.Errorf("unsupported lab node version %d", node.Version)
	}
	if node.NodeID == "" || node.Role == "" || node.ClusterID == "" || node.PrivateIdentity == "" {
		return Node{}, errors.New("lab node is incomplete")
	}
	return node, nil
}

func AddTrust(stateDir string, peer Node, endpoint string, now time.Time) error {
	return AddTrustWithEndpoints(stateDir, peer, EndpointSet{Last: strings.TrimSpace(endpoint)}, now)
}

func AddTrustWithEndpoints(stateDir string, peer Node, endpoints EndpointSet, now time.Time) error {
	if stateDir == "" {
		return errors.New("state directory is required")
	}
	if peer.NodeID == "" {
		return errors.New("peer node is required")
	}
	trust, err := LoadTrust(stateDir)
	if err != nil {
		if errors.Is(unwrapPathError(err), os.ErrNotExist) {
			trust = TrustStore{Version: 1, Peers: map[string]TrustRecord{}}
		} else {
			return err
		}
	}
	if trust.Peers == nil {
		trust.Peers = map[string]TrustRecord{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	endpoints = endpoints.Normalized()
	trust.Peers[peer.NodeID] = TrustRecord{
		Version:             1,
		NodeID:              peer.NodeID,
		Role:                peer.Role,
		ClusterID:           peer.ClusterID,
		IdentityFingerprint: peer.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:      peer.PublicIdentity,
		LastEndpoint:        endpoints.Last,
		Endpoints:           endpoints,
		TrustedAt:           now.UTC(),
	}
	_, err = private.WriteJSONAtomic(trustFile(stateDir), trust)
	return err
}

func Pair(opts PairOptions) error {
	if opts.MasterStateDir == "" || opts.WorkerStateDir == "" {
		return errors.New("master and worker state directories are required")
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	master, err := LoadNode(opts.MasterStateDir)
	if err != nil {
		return fmt.Errorf("load master lab node: %w", err)
	}
	worker, err := LoadNode(opts.WorkerStateDir)
	if err != nil {
		return fmt.Errorf("load worker lab node: %w", err)
	}
	if master.Role != RoleMaster {
		return fmt.Errorf("master state dir contains %s node", master.Role)
	}
	if worker.Role != RoleWorker {
		return fmt.Errorf("worker state dir contains %s node", worker.Role)
	}
	if master.ClusterID != worker.ClusterID {
		return fmt.Errorf("lab nodes are in different clusters: %s vs %s", master.ClusterID, worker.ClusterID)
	}
	if err := AddTrustWithEndpoints(opts.WorkerStateDir, master, EndpointSet{
		Public: opts.MasterPublicEndpoint,
		VPC:    opts.MasterVPCEndpoint,
		Last:   opts.MasterEndpoint,
	}, now); err != nil {
		return err
	}
	if opts.TrustBothWays {
		if err := AddTrustWithEndpoints(opts.MasterStateDir, worker, EndpointSet{
			Public: opts.WorkerPublicEndpoint,
			VPC:    opts.WorkerVPCEndpoint,
			Last:   opts.WorkerEndpoint,
		}, now); err != nil {
			return err
		}
	}
	return nil
}

func LoadTrust(stateDir string) (TrustStore, error) {
	var trust TrustStore
	if err := private.ReadJSON(trustFile(stateDir), &trust); err != nil {
		return TrustStore{}, err
	}
	if trust.Version == 0 {
		trust.Version = 1
	}
	if trust.Peers == nil {
		trust.Peers = map[string]TrustRecord{}
	}
	return trust, nil
}

func ListTrust(stateDir string) ([]TrustRecord, error) {
	trust, err := LoadTrust(stateDir)
	if err != nil {
		return nil, err
	}
	peers := make([]TrustRecord, 0, len(trust.Peers))
	for _, peer := range trust.Peers {
		peers = append(peers, peer)
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].NodeID < peers[j].NodeID
	})
	return peers, nil
}

func RemoveTrust(stateDir, peerNodeID string) error {
	if stateDir == "" {
		return errors.New("state directory is required")
	}
	peerNodeID = strings.TrimSpace(peerNodeID)
	if peerNodeID == "" {
		return errors.New("peer node id is required")
	}
	if peerNodeID == "." || peerNodeID == ".." || strings.ContainsAny(peerNodeID, `/\`) {
		return fmt.Errorf("invalid peer node id %q", peerNodeID)
	}
	trust, err := LoadTrust(stateDir)
	if err != nil {
		return err
	}
	delete(trust.Peers, peerNodeID)
	_, err = private.WriteJSONAtomic(trustFile(stateDir), trust)
	return err
}

func StorePaths(stateDir string) store.Paths {
	return store.Paths{
		StateDir: stateDir,
		AgentDir: filepath.Join(stateDir, "agent"),
	}
}

func Start(ctx context.Context, stateDir, listenHost string, listenPort int) (*transport.Transport, Node, error) {
	node, err := LoadNode(stateDir)
	if err != nil {
		return nil, Node{}, err
	}
	local, err := node.LocalNode()
	if err != nil {
		return nil, Node{}, err
	}
	transport, err := transport.Start(ctx, local, transport.Options{
		ListenHost:     listenHost,
		ListenUDPPort:  listenPort,
		TrustValidator: TrustValidator{StateDir: stateDir},
		PeerObserver:   store.PeerWriter{Paths: StorePaths(stateDir)},
		Enrollment:     NewEnrollmentHandler(stateDir, node),
	})
	if err != nil {
		return nil, Node{}, err
	}
	status := store.Status{
		Version:       1,
		NodeID:        node.NodeID,
		Role:          node.Role,
		Enabled:       true,
		State:         store.StateListening,
		Health:        store.HealthHealthy,
		ListenUDPPort: transport.BoundUDPPort(),
		BoundEndpoint: transport.BoundEndpoint(),
		StartedAt:     time.Now().UTC(),
		LastUpdatedAt: time.Now().UTC(),
		Message:       "secureconn lab transport listening",
	}
	_, _ = store.WriteStatus(StorePaths(stateDir), status)
	return transport, node, nil
}

func Ping(ctx context.Context, stateDir, peerNodeID, endpoint string) (time.Duration, error) {
	labTransport, _, err := Start(ctx, stateDir, "127.0.0.1", 0)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = labTransport.Close()
		_ = WriteStoppedStatus(stateDir, "secureconn lab ping transport stopped")
	}()
	return labTransport.Ping(ctx, peerNodeID, endpoint)
}

func WriteStoppedStatus(stateDir, message string) error {
	node, err := LoadNode(stateDir)
	if err != nil {
		return err
	}
	peers, err := store.ListPeers(StorePaths(stateDir))
	if err != nil {
		return err
	}
	status := store.Status{
		Version:       1,
		NodeID:        node.NodeID,
		Role:          node.Role,
		Enabled:       false,
		State:         store.StateStopped,
		Health:        store.HealthHealthy,
		LastUpdatedAt: time.Now().UTC(),
		PeerCount:     len(peers),
		Message:       message,
	}
	for _, peer := range peers {
		if peer.SessionState == store.SessionStateConnected {
			status.EstablishedPeerCount++
		}
	}
	_, err = store.WriteStatus(StorePaths(stateDir), status)
	return err
}

func Status(stateDir string) (Node, store.Status, []store.PeerObservation, []TrustRecord, error) {
	node, err := LoadNode(stateDir)
	if err != nil {
		return Node{}, store.Status{}, nil, nil, err
	}
	status, err := store.ReadStatus(StorePaths(stateDir))
	if err != nil && !errors.Is(unwrapPathError(err), os.ErrNotExist) {
		return Node{}, store.Status{}, nil, nil, err
	}
	if status.Version == 0 {
		status = store.Status{
			Version:       1,
			NodeID:        node.NodeID,
			Role:          node.Role,
			Enabled:       false,
			State:         store.StateStopped,
			Health:        store.HealthHealthy,
			LastUpdatedAt: time.Now().UTC(),
			Message:       "lab transport has not written status yet",
		}
	}
	peers, err := store.ListPeers(StorePaths(stateDir))
	if err != nil {
		return Node{}, store.Status{}, nil, nil, err
	}
	trusted, err := ListTrust(stateDir)
	if err != nil && !errors.Is(unwrapPathError(err), os.ErrNotExist) {
		return Node{}, store.Status{}, nil, nil, err
	}
	return node, status, peers, trusted, nil
}

func (n Node) LocalNode() (transport.LocalNode, error) {
	privateKey, err := base64.RawStdEncoding.DecodeString(n.PrivateIdentity)
	if err != nil {
		return transport.LocalNode{}, fmt.Errorf("decode private identity: %w", err)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return transport.LocalNode{}, fmt.Errorf("private identity has %d bytes, expected %d", len(privateKey), ed25519.PrivateKeySize)
	}
	return transport.LocalNode{
		NodeID:          n.NodeID,
		Role:            n.Role,
		ClusterID:       n.ClusterID,
		PublicIdentity:  n.PublicIdentity,
		PrivateIdentity: ed25519.PrivateKey(privateKey),
	}, nil
}

type TrustValidator struct {
	StateDir string
}

func (v TrustValidator) ValidateInitiator(peer transport.Peer) error {
	return v.validate(peer)
}

func (v TrustValidator) ValidateResponder(peer transport.Peer) error {
	return v.validate(peer)
}

func (v TrustValidator) validate(peer transport.Peer) error {
	trust, err := LoadTrust(v.StateDir)
	if err != nil {
		return err
	}
	record, ok := trust.Peers[peer.NodeID]
	if !ok {
		return fmt.Errorf("peer %s is not trusted", peer.NodeID)
	}
	if record.Role != peer.Role || record.ClusterID != peer.ClusterID || record.IdentityFingerprint != peer.IdentityFingerprint {
		return fmt.Errorf("peer %s identity does not match trust record", peer.NodeID)
	}
	if record.PublicIdentity.PublicKey != peer.PublicIdentity.PublicKey {
		return fmt.Errorf("peer %s public key does not match trust record", peer.NodeID)
	}
	return nil
}

func Endpoint(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func (e EndpointSet) Normalized() EndpointSet {
	return EndpointSet{
		Public: strings.TrimSpace(e.Public),
		VPC:    strings.TrimSpace(e.VPC),
		Last:   strings.TrimSpace(e.Last),
	}
}

func (e EndpointSet) IsZero() bool {
	e = e.Normalized()
	return e.Public == "" && e.VPC == "" && e.Last == ""
}

func randomID(size int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz234567"
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "node"
	}
	for i, value := range data {
		data[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(data)
}

func nodeFile(stateDir string) string {
	return filepath.Join(stateDir, "lab", "node.json")
}

func trustFile(stateDir string) string {
	return filepath.Join(stateDir, "lab", "trust.json")
}

func unwrapPathError(err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}
