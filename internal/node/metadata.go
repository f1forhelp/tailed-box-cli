package node

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

type Metadata struct {
	Version             int       `json:"version"`
	NodeID              string    `json:"node_id"`
	Role                string    `json:"role"`
	ClusterID           string    `json:"cluster_id,omitempty"`
	ClusterName         string    `json:"cluster_name,omitempty"`
	InitializedAt       time.Time `json:"initialized_at"`
	IdentityAlgorithm   string    `json:"identity_algorithm,omitempty"`
	IdentityFingerprint string    `json:"identity_fingerprint,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

type EnsureResult struct {
	Changed  bool
	RoleDir  string
	Path     string
	Metadata Metadata
}

func EnsureMetadata(cfg *config.Config, now time.Time) (EnsureResult, error) {
	if cfg == nil {
		return EnsureResult{}, errors.New("config is nil")
	}
	roleDir := config.RoleStateDir(cfg)
	if roleDir == "" {
		return EnsureResult{}, errors.New("node role is required before metadata initialization")
	}
	if err := secrets.EnsurePrivateDir(roleDir); err != nil {
		return EnsureResult{}, err
	}

	createdAt := now.UTC()
	var existing Metadata
	if err := secrets.ReadJSON(cfg.Paths.NodeMetadataFile, &existing); err == nil {
		if existing.NodeID != cfg.Node.ID {
			return EnsureResult{}, fmt.Errorf("node metadata belongs to node %s, expected %s", existing.NodeID, cfg.Node.ID)
		}
		if existing.Role != cfg.Node.Role {
			return EnsureResult{}, fmt.Errorf("node metadata belongs to role %s, expected %s", existing.Role, cfg.Node.Role)
		}
		if !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return EnsureResult{}, err
	}

	metadata := Metadata{
		Version:             1,
		NodeID:              cfg.Node.ID,
		Role:                cfg.Node.Role,
		ClusterID:           cfg.Cluster.ID,
		ClusterName:         cfg.Cluster.Name,
		InitializedAt:       cfg.Node.InitializedAt,
		IdentityAlgorithm:   cfg.Node.Identity.Algorithm,
		IdentityFingerprint: cfg.Node.Identity.PublicKeyFingerprint,
		CreatedAt:           createdAt,
	}
	changed, err := secrets.WriteJSONAtomic(cfg.Paths.NodeMetadataFile, metadata)
	if err != nil {
		return EnsureResult{}, err
	}
	return EnsureResult{Changed: changed, RoleDir: roleDir, Path: cfg.Paths.NodeMetadataFile, Metadata: metadata}, nil
}
