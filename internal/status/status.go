package status

import (
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
)

type NodeHealth string

const (
	HealthHealthy  NodeHealth = "healthy"
	HealthDegraded NodeHealth = "degraded"
)

type MeshState string

const (
	MeshNotConfigured MeshState = "not_configured"
	MeshDisconnected  MeshState = "disconnected"
)

type NodeStatus struct {
	NodeID       string     `json:"node_id"`
	Role         string     `json:"role"`
	Reachability string     `json:"reachability"`
	MeshState    MeshState  `json:"mesh_state"`
	LastSeen     time.Time  `json:"last_seen"`
	Health       NodeHealth `json:"health"`
}

type MasterStatus struct {
	Current    NodeStatus   `json:"current"`
	KnownNodes []NodeStatus `json:"known_nodes"`
	Summary    Summary      `json:"summary"`
}

type Summary struct {
	Masters  int `json:"masters"`
	Workers  int `json:"workers"`
	Healthy  int `json:"healthy"`
	Degraded int `json:"degraded"`
}

type WorkerStatus struct {
	NodeID                   string     `json:"node_id"`
	Role                     string     `json:"role"`
	Initialized              bool       `json:"initialized"`
	JoinedToMasterCluster    bool       `json:"joined_to_master_cluster"`
	ConnectedToMasterCluster bool       `json:"connected_to_master_cluster"`
	Authenticated            bool       `json:"authenticated"`
	MeshReachable            bool       `json:"mesh_reachable"`
	MeshState                MeshState  `json:"mesh_state"`
	Health                   NodeHealth `json:"health"`
}

type LocalStatus struct {
	NodeID      string     `json:"node_id"`
	Role        string     `json:"role"`
	Initialized bool       `json:"initialized"`
	ConfigFile  string     `json:"config_file"`
	StateDir    string     `json:"state_dir"`
	LogFile     string     `json:"log_file"`
	Health      NodeHealth `json:"health"`
}

func ForMaster(cfg *config.Config, now time.Time) MasterStatus {
	current := baseNodeStatus(cfg, now)
	nodes := []NodeStatus{current}
	return MasterStatus{
		Current:    current,
		KnownNodes: nodes,
		Summary:    summarize(nodes),
	}
}

func ForWorker(cfg *config.Config) WorkerStatus {
	initialized := cfg.Node.Role == config.RoleWorker
	health := HealthDegraded
	if initialized {
		health = HealthDegraded
	}
	return WorkerStatus{
		NodeID:                   nodeIDOrUnassigned(cfg),
		Role:                     roleOrUninitialized(cfg),
		Initialized:              initialized,
		JoinedToMasterCluster:    false,
		ConnectedToMasterCluster: false,
		Authenticated:            false,
		MeshReachable:            false,
		MeshState:                MeshNotConfigured,
		Health:                   health,
	}
}

func ForLocal(cfg *config.Config) LocalStatus {
	initialized := cfg.Node.Role != ""
	health := HealthDegraded
	return LocalStatus{
		NodeID:      nodeIDOrUnassigned(cfg),
		Role:        roleOrUninitialized(cfg),
		Initialized: initialized,
		ConfigFile:  cfg.Paths.ConfigFile,
		StateDir:    cfg.Paths.StateDir,
		LogFile:     cfg.Paths.LogFile,
		Health:      health,
	}
}

func baseNodeStatus(cfg *config.Config, now time.Time) NodeStatus {
	role := roleOrUninitialized(cfg)
	return NodeStatus{
		NodeID:       nodeIDOrUnassigned(cfg),
		Role:         role,
		Reachability: "local",
		MeshState:    MeshNotConfigured,
		LastSeen:     now.UTC(),
		Health:       HealthDegraded,
	}
}

func summarize(nodes []NodeStatus) Summary {
	var summary Summary
	for _, node := range nodes {
		switch node.Role {
		case config.RoleMaster:
			summary.Masters++
		case config.RoleWorker:
			summary.Workers++
		}
		switch node.Health {
		case HealthHealthy:
			summary.Healthy++
		default:
			summary.Degraded++
		}
	}
	return summary
}

func nodeIDOrUnassigned(cfg *config.Config) string {
	if cfg.Node.ID == "" {
		return "unassigned"
	}
	return cfg.Node.ID
}

func roleOrUninitialized(cfg *config.Config) string {
	if cfg.Node.Role == "" {
		return "uninitialized"
	}
	return cfg.Node.Role
}
