package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	secureidentity "github.com/tailedbox/link/identity"
	"github.com/tailedbox/link/store"
	"github.com/tailedbox/link/transport"
	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/identity"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const trustStateTrusted = "trusted"

type trustedNodeRecord struct {
	NodeID                  string    `json:"node_id"`
	Role                    string    `json:"role"`
	IdentityFingerprint     string    `json:"identity_fingerprint"`
	PublicKey               string    `json:"public_key"`
	ClusterID               string    `json:"cluster_id"`
	TrustState              string    `json:"trust_state"`
	ReconnectLeaseExpiresAt time.Time `json:"reconnect_lease_expires_at"`
}

type joinedClusterRecord struct {
	ClusterID                 string `json:"cluster_id"`
	MasterNodeID              string `json:"master_node_id"`
	MasterIdentityFingerprint string `json:"master_identity_fingerprint"`
}

type tailedboxTrustValidator struct {
	cfg *config.Config
	now func() time.Time
}

func securePaths(paths config.Paths) store.Paths {
	return store.Paths{
		StateDir: paths.StateDir,
		AgentDir: paths.AgentDir,
	}
}

func localNodeFromConfig(cfg *config.Config) (transport.LocalNode, error) {
	privateKey, _, err := identity.LoadPrivateKey(cfg.Paths.IdentityPrivateKeyFile)
	if err != nil {
		return transport.LocalNode{}, fmt.Errorf("load local mesh identity private key: %w", err)
	}
	publicIdentity, err := identity.LoadPublicIdentity(cfg.Paths.IdentityPublicKeyFile)
	if err != nil {
		return transport.LocalNode{}, fmt.Errorf("load local mesh public identity: %w", err)
	}
	return transport.LocalNode{
		NodeID:          cfg.Node.ID,
		Role:            cfg.Node.Role,
		ClusterID:       cfg.Cluster.ID,
		PublicIdentity:  convertPublicIdentity(publicIdentity),
		PrivateIdentity: privateKey,
	}, nil
}

func convertPublicIdentity(value identity.PublicIdentity) secureidentity.PublicIdentity {
	return secureidentity.PublicIdentity{
		Version:              value.Version,
		NodeID:               value.NodeID,
		Algorithm:            value.Algorithm,
		PublicKey:            value.PublicKey,
		PublicKeyFingerprint: value.PublicKeyFingerprint,
		CreatedAt:            value.CreatedAt,
	}
}

func (v tailedboxTrustValidator) ValidateInitiator(peer transport.Peer) error {
	if err := v.validateCluster(peer); err != nil {
		return err
	}
	switch v.cfg.Node.Role {
	case config.RoleMaster:
		return v.validateTrustedWorker(peer)
	case config.RoleWorker:
		return v.validatePinnedMaster(peer)
	default:
		return fmt.Errorf("unsupported local mesh role %q", v.cfg.Node.Role)
	}
}

func (v tailedboxTrustValidator) ValidateResponder(peer transport.Peer) error {
	if err := v.validateCluster(peer); err != nil {
		return err
	}
	switch v.cfg.Node.Role {
	case config.RoleWorker:
		return v.validatePinnedMaster(peer)
	case config.RoleMaster:
		return v.validateTrustedWorker(peer)
	default:
		return fmt.Errorf("unsupported local mesh role %q", v.cfg.Node.Role)
	}
}

func (v tailedboxTrustValidator) validateCluster(peer transport.Peer) error {
	if peer.ClusterID != v.cfg.Cluster.ID {
		return fmt.Errorf("mesh peer cluster %s does not match local cluster %s", peer.ClusterID, v.cfg.Cluster.ID)
	}
	return nil
}

func (v tailedboxTrustValidator) validateTrustedWorker(peer transport.Peer) error {
	trusted, err := readTrustedNode(v.cfg, peer.NodeID)
	if err != nil {
		return fmt.Errorf("mesh peer is not trusted: %w", err)
	}
	if trusted.TrustState != trustStateTrusted {
		return fmt.Errorf("mesh peer %s is not trusted", peer.NodeID)
	}
	if trusted.Role != peer.Role || trusted.ClusterID != peer.ClusterID || trusted.IdentityFingerprint != peer.IdentityFingerprint || trusted.PublicKey != peer.PublicIdentity.PublicKey {
		return errors.New("mesh peer identity does not match trusted-node record")
	}
	if !trusted.ReconnectLeaseExpiresAt.IsZero() && !v.nowUTC().Before(trusted.ReconnectLeaseExpiresAt) {
		return fmt.Errorf("mesh peer reconnect lease has expired")
	}
	return nil
}

func (v tailedboxTrustValidator) validatePinnedMaster(peer transport.Peer) error {
	joined, err := readJoinedCluster(v.cfg)
	if err != nil {
		return err
	}
	if peer.NodeID != joined.MasterNodeID || peer.IdentityFingerprint != joined.MasterIdentityFingerprint || peer.ClusterID != joined.ClusterID || peer.Role != config.RoleMaster {
		return errors.New("mesh peer does not match pinned master identity")
	}
	return nil
}

func (v tailedboxTrustValidator) nowUTC() time.Time {
	if v.now == nil {
		return time.Now().UTC()
	}
	return v.now().UTC()
}

func readTrustedNode(cfg *config.Config, nodeID string) (trustedNodeRecord, error) {
	if nodeID == "" || nodeID == "." || nodeID == ".." || strings.ContainsAny(nodeID, `/\`) {
		return trustedNodeRecord{}, fmt.Errorf("invalid trusted node id %q", nodeID)
	}
	var record trustedNodeRecord
	if err := secrets.ReadJSON(filepath.Join(cfg.Paths.TrustedNodesDir, nodeID+".json"), &record); err != nil {
		return trustedNodeRecord{}, err
	}
	return record, nil
}

func readJoinedCluster(cfg *config.Config) (joinedClusterRecord, error) {
	var record joinedClusterRecord
	if err := secrets.ReadJSON(cfg.Paths.JoinedClusterFile, &record); err != nil {
		return joinedClusterRecord{}, err
	}
	return record, nil
}
