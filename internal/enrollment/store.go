package enrollment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	JoinCodeStateActive  = "active"
	JoinCodeStateUsed    = "used"
	JoinCodeStateExpired = "expired"
	TrustStateTrusted    = "trusted"
)

type JoinCodeRecord struct {
	Version                 int        `json:"version"`
	ID                      string     `json:"id"`
	AllowedRole             string     `json:"allowed_role"`
	ClusterID               string     `json:"cluster_id"`
	ClusterName             string     `json:"cluster_name,omitempty"`
	IssuerNodeID            string     `json:"issuer_node_id"`
	IssuerFingerprint       string     `json:"issuer_fingerprint"`
	SecretHash              string     `json:"secret_hash"`
	SecretHashAlgorithm     string     `json:"secret_hash_algorithm"`
	State                   string     `json:"state"`
	CreatedAt               time.Time  `json:"created_at"`
	ExpiresAt               time.Time  `json:"expires_at"`
	UsedAt                  *time.Time `json:"used_at,omitempty"`
	UsedByNodeID            string     `json:"used_by_node_id,omitempty"`
	UsedByFingerprint       string     `json:"used_by_fingerprint,omitempty"`
	ReconnectWindow         string     `json:"reconnect_window"`
	ReconnectLeaseExpiresAt *time.Time `json:"reconnect_lease_expires_at,omitempty"`
}

type TrustedNode struct {
	Version                 int       `json:"version"`
	NodeID                  string    `json:"node_id"`
	Role                    string    `json:"role"`
	IdentityAlgorithm       string    `json:"identity_algorithm"`
	IdentityFingerprint     string    `json:"identity_fingerprint"`
	PublicKey               string    `json:"public_key"`
	ClusterID               string    `json:"cluster_id"`
	ClusterName             string    `json:"cluster_name,omitempty"`
	JoinCodeID              string    `json:"join_code_id"`
	TrustState              string    `json:"trust_state"`
	JoinedAt                time.Time `json:"joined_at"`
	ReconnectLeaseExpiresAt time.Time `json:"reconnect_lease_expires_at"`
}

type JoinedCluster struct {
	Version                   int       `json:"version"`
	NodeID                    string    `json:"node_id"`
	Role                      string    `json:"role"`
	ClusterID                 string    `json:"cluster_id"`
	ClusterName               string    `json:"cluster_name,omitempty"`
	MasterNodeID              string    `json:"master_node_id"`
	MasterIdentityFingerprint string    `json:"master_identity_fingerprint"`
	JoinCodeID                string    `json:"join_code_id"`
	JoinedAt                  time.Time `json:"joined_at"`
	ReconnectLeaseExpiresAt   time.Time `json:"reconnect_lease_expires_at"`
}

func ensureStoreDirs(paths config.Paths) error {
	for _, dir := range []string{paths.EnrollmentDir, paths.JoinCodesDir, paths.TrustedNodesDir} {
		if err := secrets.EnsurePrivateDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func WriteJoinCodeRecord(paths config.Paths, record JoinCodeRecord) error {
	if err := ensureStoreDirs(paths); err != nil {
		return err
	}
	if _, err := secrets.WriteJSONAtomic(joinCodeRecordPath(paths, record.ID), record); err != nil {
		return err
	}
	return nil
}

func ReadJoinCodeRecord(paths config.Paths, id string) (JoinCodeRecord, error) {
	var record JoinCodeRecord
	if err := secrets.ReadJSON(joinCodeRecordPath(paths, id), &record); err != nil {
		return JoinCodeRecord{}, err
	}
	return record, nil
}

func WriteTrustedNode(paths config.Paths, node TrustedNode) error {
	if err := ensureStoreDirs(paths); err != nil {
		return err
	}
	if _, err := secrets.WriteJSONAtomic(trustedNodePath(paths, node.NodeID), node); err != nil {
		return err
	}
	return nil
}

func ReadTrustedNode(paths config.Paths, nodeID string) (TrustedNode, error) {
	var node TrustedNode
	if err := secrets.ReadJSON(trustedNodePath(paths, nodeID), &node); err != nil {
		return TrustedNode{}, err
	}
	return node, nil
}

func TrustedNodeExists(paths config.Paths, nodeID string) bool {
	return secrets.Exists(trustedNodePath(paths, nodeID))
}

func WriteJoinedCluster(path string, joined JoinedCluster) error {
	if _, err := secrets.WriteJSONAtomic(path, joined); err != nil {
		return err
	}
	return nil
}

func ReadJoinedCluster(path string) (JoinedCluster, error) {
	var joined JoinedCluster
	if err := secrets.ReadJSON(path, &joined); err != nil {
		return JoinedCluster{}, err
	}
	return joined, nil
}

func ListTrustedNodes(paths config.Paths) ([]TrustedNode, error) {
	entries, err := os.ReadDir(paths.TrustedNodesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read trusted nodes directory: %w", err)
	}
	nodes := make([]TrustedNode, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var node TrustedNode
		if err := secrets.ReadJSON(filepath.Join(paths.TrustedNodesDir, entry.Name()), &node); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeID < nodes[j].NodeID
	})
	return nodes, nil
}

func joinCodeRecordPath(paths config.Paths, id string) string {
	return filepath.Join(paths.JoinCodesDir, id+".json")
}

func trustedNodePath(paths config.Paths, nodeID string) string {
	return filepath.Join(paths.TrustedNodesDir, nodeID+".json")
}
