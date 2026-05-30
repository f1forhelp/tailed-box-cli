package agent

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

type Config struct {
	Version   int        `json:"version"`
	NodeID    string     `json:"node_id"`
	Role      string     `json:"role"`
	StateDir  string     `json:"state_dir"`
	LogFile   string     `json:"log_file"`
	CreatedAt time.Time  `json:"created_at"`
	Mesh      MeshConfig `json:"mesh"`
}

type MeshConfig struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	ListenUDPPort int    `json:"listen_udp_port"`
}

type EnsureResult struct {
	Changed bool
	Path    string
	Config  Config
}

func EnsureConfig(cfg *config.Config, now time.Time) (EnsureResult, error) {
	if cfg == nil {
		return EnsureResult{}, errors.New("config is nil")
	}
	if cfg.Node.ID == "" || cfg.Node.Role == "" {
		return EnsureResult{}, errors.New("node id and role are required before agent config initialization")
	}

	createdAt := now.UTC()
	var existing Config
	if err := secrets.ReadJSON(cfg.Paths.AgentConfigFile, &existing); err == nil {
		if existing.NodeID != cfg.Node.ID {
			return EnsureResult{}, fmt.Errorf("agent config belongs to node %s, expected %s", existing.NodeID, cfg.Node.ID)
		}
		if existing.Role != cfg.Node.Role {
			return EnsureResult{}, fmt.Errorf("agent config belongs to role %s, expected %s", existing.Role, cfg.Node.Role)
		}
		if !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return EnsureResult{}, err
	}

	agentConfig := Config{
		Version:   1,
		NodeID:    cfg.Node.ID,
		Role:      cfg.Node.Role,
		StateDir:  cfg.Paths.StateDir,
		LogFile:   cfg.Paths.LogFile,
		CreatedAt: createdAt,
		Mesh: MeshConfig{
			Enabled:       false,
			Provider:      "tailedbox-mesh",
			ListenUDPPort: 0,
		},
	}
	changed, err := secrets.WriteJSONAtomic(cfg.Paths.AgentConfigFile, agentConfig)
	if err != nil {
		return EnsureResult{}, err
	}
	return EnsureResult{Changed: changed, Path: cfg.Paths.AgentConfigFile, Config: agentConfig}, nil
}
