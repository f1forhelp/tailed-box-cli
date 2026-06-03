package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	meshservice "github.com/tailedbox/tailedbox/internal/mesh/service"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	StateNotInitialized = "not_initialized"
	StateStopped        = "stopped"
	StateRunning        = "running"
	StateStale          = "stale"

	HealthHealthy  = "healthy"
	HealthDegraded = "degraded"

	DefaultHeartbeatInterval = 10 * time.Second
)

type Status struct {
	Version              int       `json:"version"`
	NodeID               string    `json:"node_id"`
	Role                 string    `json:"role"`
	State                string    `json:"state"`
	Health               string    `json:"health"`
	Running              bool      `json:"running"`
	PID                  int       `json:"pid,omitempty"`
	StartedAt            time.Time `json:"started_at,omitempty"`
	LastHeartbeatAt      time.Time `json:"last_heartbeat_at,omitempty"`
	UptimeSeconds        int64     `json:"uptime_seconds"`
	HeartbeatAgeSeconds  int64     `json:"heartbeat_age_seconds"`
	MemoryAllocBytes     uint64    `json:"memory_alloc_bytes"`
	MemorySysBytes       uint64    `json:"memory_sys_bytes"`
	Goroutines           int       `json:"goroutines"`
	ConfigFile           string    `json:"config_file"`
	StateDir             string    `json:"state_dir"`
	LogFile              string    `json:"log_file"`
	AgentConfigFile      string    `json:"agent_config_file"`
	AgentStatusFile      string    `json:"agent_status_file"`
	SystemdServiceName   string    `json:"systemd_service_name"`
	SystemdUnitPath      string    `json:"systemd_unit_path"`
	Message              string    `json:"message,omitempty"`
	HeartbeatIntervalSec int64     `json:"heartbeat_interval_seconds"`
}

type RunOptions struct {
	HeartbeatInterval time.Duration
	Logger            *slog.Logger
	Now               func() time.Time
}

func Run(ctx context.Context, cfg *config.Config, opts RunOptions) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.Node.ID == "" || cfg.Node.Role == "" {
		return errors.New("node must be initialized before running the agent")
	}
	agentConfigResult, err := EnsureConfig(cfg, nowFrom(opts.Now))
	if err != nil {
		return fmt.Errorf("ensure agent config: %w", err)
	}
	meshSvc, err := meshservice.Start(ctx, cfg, meshservice.MeshConfig{
		Enabled:       agentConfigResult.Config.Mesh.Enabled,
		Provider:      agentConfigResult.Config.Mesh.Provider,
		ListenUDPPort: agentConfigResult.Config.Mesh.ListenUDPPort,
	}, meshservice.Options{
		Logger: opts.Logger,
		Now:    opts.Now,
	})
	if err != nil {
		return fmt.Errorf("start mesh service: %w", err)
	}
	defer meshSvc.Close()

	interval := opts.HeartbeatInterval
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	now := nowFunc(opts.Now)
	startedAt := now()
	pid := os.Getpid()

	write := func(state, health, message string) error {
		status := buildRuntimeStatus(cfg, state, health, pid, startedAt, now(), interval, message)
		return WriteStatus(cfg, status)
	}

	if opts.Logger != nil {
		opts.Logger.InfoContext(ctx, "agent started", "node_id", cfg.Node.ID, "role", cfg.Node.Role, "heartbeat_interval", interval.String())
	}
	if err := write(StateRunning, HealthHealthy, "agent running"); err != nil {
		return err
	}
	if err := meshSvc.RefreshStatus(); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if opts.Logger != nil {
				opts.Logger.InfoContext(context.Background(), "agent stopped", "node_id", cfg.Node.ID, "role", cfg.Node.Role)
			}
			if err := meshSvc.RefreshStatus(); err != nil {
				return err
			}
			if err := write(StateStopped, HealthDegraded, "agent stopped"); err != nil {
				return err
			}
			return nil
		case <-ticker.C:
			if opts.Logger != nil {
				opts.Logger.DebugContext(ctx, "agent heartbeat", "node_id", cfg.Node.ID)
			}
			if err := meshSvc.RefreshStatus(); err != nil {
				return err
			}
			if err := write(StateRunning, HealthHealthy, "agent running"); err != nil {
				return err
			}
		}
	}
}

func WriteStatus(cfg *config.Config, status Status) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	_, err := secrets.WriteJSONAtomic(cfg.Paths.AgentStatusFile, status)
	return err
}

func ReadStatus(cfg *config.Config, now time.Time) (Status, error) {
	if cfg == nil {
		return Status{}, errors.New("config is nil")
	}
	if cfg.Node.ID == "" || cfg.Node.Role == "" {
		return baseStatus(cfg, StateNotInitialized, HealthDegraded, "node is not initialized"), nil
	}
	var value Status
	if err := secrets.ReadJSON(cfg.Paths.AgentStatusFile, &value); err == nil {
		return normalizeStatus(cfg, value, now.UTC()), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Status{}, err
	}
	return baseStatus(cfg, StateStopped, HealthDegraded, "agent has not written a heartbeat yet"), nil
}

func buildRuntimeStatus(cfg *config.Config, state, health string, pid int, startedAt, heartbeatAt time.Time, interval time.Duration, message string) Status {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	uptimeSeconds := int64(0)
	if !startedAt.IsZero() {
		uptimeSeconds = int64(heartbeatAt.Sub(startedAt).Seconds())
	}
	status := baseStatus(cfg, state, health, message)
	status.Running = state == StateRunning
	status.PID = pid
	status.StartedAt = startedAt.UTC()
	status.LastHeartbeatAt = heartbeatAt.UTC()
	status.UptimeSeconds = uptimeSeconds
	status.HeartbeatAgeSeconds = 0
	status.MemoryAllocBytes = mem.Alloc
	status.MemorySysBytes = mem.Sys
	status.Goroutines = runtime.NumGoroutine()
	status.HeartbeatIntervalSec = int64(interval.Seconds())
	return status
}

func normalizeStatus(cfg *config.Config, value Status, now time.Time) Status {
	if value.Version == 0 {
		value.Version = 1
	}
	value.NodeID = cfg.Node.ID
	value.Role = cfg.Node.Role
	value.ConfigFile = cfg.Paths.ConfigFile
	value.StateDir = cfg.Paths.StateDir
	value.LogFile = cfg.Paths.LogFile
	value.AgentConfigFile = cfg.Paths.AgentConfigFile
	value.AgentStatusFile = cfg.Paths.AgentStatusFile
	value.SystemdServiceName = SystemdServiceName
	value.SystemdUnitPath = DefaultSystemdUnitPath
	if value.LastHeartbeatAt.IsZero() {
		value.Running = false
		value.Health = HealthDegraded
		if value.State == "" {
			value.State = StateStopped
		}
		return value
	}
	age := now.Sub(value.LastHeartbeatAt.UTC())
	if age < 0 {
		age = 0
	}
	value.HeartbeatAgeSeconds = int64(age.Seconds())
	staleAfter := staleAfter(value.HeartbeatIntervalSec)
	if value.State == StateRunning && age > staleAfter {
		value.State = StateStale
		value.Health = HealthDegraded
		value.Running = false
		value.Message = "last heartbeat is stale"
	}
	return value
}

func baseStatus(cfg *config.Config, state, health, message string) Status {
	return Status{
		Version:              1,
		NodeID:               cfg.Node.ID,
		Role:                 cfg.Node.Role,
		State:                state,
		Health:               health,
		Running:              state == StateRunning,
		ConfigFile:           cfg.Paths.ConfigFile,
		StateDir:             cfg.Paths.StateDir,
		LogFile:              cfg.Paths.LogFile,
		AgentConfigFile:      cfg.Paths.AgentConfigFile,
		AgentStatusFile:      cfg.Paths.AgentStatusFile,
		SystemdServiceName:   SystemdServiceName,
		SystemdUnitPath:      DefaultSystemdUnitPath,
		Message:              message,
		HeartbeatIntervalSec: int64(DefaultHeartbeatInterval.Seconds()),
	}
}

func staleAfter(intervalSeconds int64) time.Duration {
	if intervalSeconds <= 0 {
		return DefaultHeartbeatInterval * 3
	}
	return time.Duration(intervalSeconds) * time.Second * 3
}

func nowFunc(fn func() time.Time) func() time.Time {
	if fn != nil {
		return func() time.Time {
			return fn().UTC()
		}
	}
	return func() time.Time {
		return time.Now().UTC()
	}
}

func nowFrom(fn func() time.Time) time.Time {
	return nowFunc(fn)()
}
