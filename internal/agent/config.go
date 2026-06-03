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

const (
	DefaultMeshProvider      = "tailedbox-mesh"
	DefaultMasterMeshUDPPort = 41677
)

type EnsureResult struct {
	Changed bool
	Path    string
	Config  Config
}

type MeshUpdateOptions struct {
	Enabled       bool
	ListenUDPPort int
	Now           time.Time
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
		Mesh:      defaultMeshConfig(cfg.Node.Role),
	}
	if existing.NodeID != "" {
		agentConfig.Mesh = normalizeMeshConfig(existing.Mesh, cfg.Node.Role)
	}
	changed, err := secrets.WriteJSONAtomic(cfg.Paths.AgentConfigFile, agentConfig)
	if err != nil {
		return EnsureResult{}, err
	}
	return EnsureResult{Changed: changed, Path: cfg.Paths.AgentConfigFile, Config: agentConfig}, nil
}

func ReadConfig(cfg *config.Config) (Config, error) {
	if cfg == nil {
		return Config{}, errors.New("config is nil")
	}
	var value Config
	if err := secrets.ReadJSON(cfg.Paths.AgentConfigFile, &value); err != nil {
		return Config{}, err
	}
	value.Mesh = normalizeMeshConfig(value.Mesh, cfg.Node.Role)
	return value, nil
}

func UpdateMeshConfig(cfg *config.Config, opts MeshUpdateOptions) (EnsureResult, error) {
	if cfg == nil {
		return EnsureResult{}, errors.New("config is nil")
	}
	if opts.ListenUDPPort < 0 || opts.ListenUDPPort > 65535 {
		return EnsureResult{}, fmt.Errorf("listen UDP port must be between 0 and 65535")
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := EnsureConfig(cfg, now)
	if err != nil {
		return EnsureResult{}, err
	}
	agentConfig := result.Config
	agentConfig.Mesh.Enabled = opts.Enabled
	if opts.ListenUDPPort != 0 {
		agentConfig.Mesh.ListenUDPPort = opts.ListenUDPPort
	}
	agentConfig.Mesh = normalizeMeshConfig(agentConfig.Mesh, cfg.Node.Role)
	changed, err := secrets.WriteJSONAtomic(cfg.Paths.AgentConfigFile, agentConfig)
	if err != nil {
		return EnsureResult{}, err
	}
	return EnsureResult{Changed: changed, Path: cfg.Paths.AgentConfigFile, Config: agentConfig}, nil
}

func defaultMeshConfig(role string) MeshConfig {
	mesh := MeshConfig{
		Enabled:       false,
		Provider:      DefaultMeshProvider,
		ListenUDPPort: 0,
	}
	if role == config.RoleMaster {
		mesh.ListenUDPPort = DefaultMasterMeshUDPPort
	}
	return mesh
}

func normalizeMeshConfig(mesh MeshConfig, role string) MeshConfig {
	defaults := defaultMeshConfig(role)
	if mesh.Provider == "" {
		mesh.Provider = defaults.Provider
	}
	if mesh.ListenUDPPort == 0 && role == config.RoleMaster {
		mesh.ListenUDPPort = defaults.ListenUDPPort
	}
	return mesh
}
