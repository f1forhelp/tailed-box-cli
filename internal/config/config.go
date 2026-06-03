package config

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RoleMaster = "master"
	RoleWorker = "worker"
)

type LoadOptions struct {
	ConfigPath string
	StateDir   string
	LogDir     string
}

type Config struct {
	Version int           `json:"version"`
	Paths   Paths         `json:"paths"`
	Node    NodeConfig    `json:"node"`
	Cluster ClusterConfig `json:"cluster,omitempty"`
	Logging LoggingConfig `json:"logging"`
}

type Paths struct {
	ConfigFile             string `json:"config_file"`
	StateDir               string `json:"state_dir"`
	LogDir                 string `json:"log_dir"`
	LogFile                string `json:"log_file"`
	SecretsDir             string `json:"secrets_dir"`
	AgentDir               string `json:"agent_dir"`
	EnrollmentDir          string `json:"enrollment_dir"`
	JoinCodesDir           string `json:"join_codes_dir"`
	TrustedNodesDir        string `json:"trusted_nodes_dir"`
	AuditDir               string `json:"audit_dir"`
	AuditLogFile           string `json:"audit_log_file"`
	NodeMetadataFile       string `json:"node_metadata_file"`
	AgentConfigFile        string `json:"agent_config_file"`
	AgentStatusFile        string `json:"agent_status_file"`
	JoinedClusterFile      string `json:"joined_cluster_file"`
	IdentityPrivateKeyFile string `json:"identity_private_key_file"`
	IdentityPublicKeyFile  string `json:"identity_public_key_file"`
}

type NodeConfig struct {
	ID            string         `json:"id,omitempty"`
	Role          string         `json:"role,omitempty"`
	InitializedAt time.Time      `json:"initialized_at,omitempty"`
	Identity      IdentityConfig `json:"identity,omitempty"`
}

type IdentityConfig struct {
	Algorithm            string    `json:"algorithm,omitempty"`
	PublicKeyFingerprint string    `json:"public_key_fingerprint,omitempty"`
	PublicKeyFile        string    `json:"public_key_file,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
}

type ClusterConfig struct {
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name,omitempty"`
	MasterEndpoints []string `json:"master_endpoints,omitempty"`
}

type LoggingConfig struct {
	DebugLogsEnabled bool `json:"debug_logs_enabled"`
}

func Load(opts LoadOptions) (*Config, error) {
	paths, err := ResolvePaths(opts)
	if err != nil {
		return nil, err
	}

	cfg := Default(paths)
	if _, err := os.Stat(paths.ConfigFile); err == nil {
		data, err := os.ReadFile(paths.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
		cfg.Paths = paths
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat config: %w", err)
	}

	if err := EnsureRuntimeDirs(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Default(paths Paths) *Config {
	return &Config{
		Version: 1,
		Paths:   paths,
		Node:    NodeConfig{},
		Logging: LoggingConfig{DebugLogsEnabled: false},
	}
}

func ResolvePaths(opts LoadOptions) (Paths, error) {
	configFile := firstNonEmpty(opts.ConfigPath, os.Getenv("TAILEDBOX_CONFIG"))
	if configFile == "" {
		configDir, err := defaultConfigDir()
		if err != nil {
			return Paths{}, err
		}
		configFile = filepath.Join(configDir, "config.json")
	}

	stateDir := firstNonEmpty(opts.StateDir, os.Getenv("TAILEDBOX_STATE_DIR"))
	if stateDir == "" {
		dir, err := defaultStateDir()
		if err != nil {
			return Paths{}, err
		}
		stateDir = dir
	}

	logDir := firstNonEmpty(opts.LogDir, os.Getenv("TAILEDBOX_LOG_DIR"))
	if logDir == "" {
		logDir = filepath.Join(stateDir, "logs")
	}

	cleanStateDir := filepath.Clean(stateDir)
	cleanLogDir := filepath.Clean(logDir)
	secretsDir := filepath.Join(cleanStateDir, "secrets")
	agentDir := filepath.Join(cleanStateDir, "agent")
	enrollmentDir := filepath.Join(cleanStateDir, "enrollment")
	auditDir := filepath.Join(cleanStateDir, "audit")

	return Paths{
		ConfigFile:             filepath.Clean(configFile),
		StateDir:               cleanStateDir,
		LogDir:                 cleanLogDir,
		LogFile:                filepath.Join(cleanLogDir, "tailedbox.log.jsonl"),
		SecretsDir:             secretsDir,
		AgentDir:               agentDir,
		EnrollmentDir:          enrollmentDir,
		JoinCodesDir:           filepath.Join(enrollmentDir, "join-codes"),
		TrustedNodesDir:        filepath.Join(enrollmentDir, "trusted-nodes"),
		AuditDir:               auditDir,
		AuditLogFile:           filepath.Join(auditDir, "events.jsonl"),
		NodeMetadataFile:       filepath.Join(cleanStateDir, "node.json"),
		AgentConfigFile:        filepath.Join(agentDir, "config.json"),
		AgentStatusFile:        filepath.Join(agentDir, "status.json"),
		JoinedClusterFile:      filepath.Join(cleanStateDir, "joined_cluster.json"),
		IdentityPrivateKeyFile: filepath.Join(secretsDir, "node_identity_ed25519.pem"),
		IdentityPublicKeyFile:  filepath.Join(cleanStateDir, "node_identity_public.json"),
	}, nil
}

func Save(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if cfg.Paths.ConfigFile == "" {
		return errors.New("config path is empty")
	}
	if err := EnsureRuntimeDirs(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Paths.ConfigFile), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(cfg.Paths.ConfigFile), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create config temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("secure config temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write config temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config temp file: %w", err)
	}
	if err := os.Rename(tmpName, cfg.Paths.ConfigFile); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	if err := os.Chmod(cfg.Paths.ConfigFile, 0o600); err != nil {
		return fmt.Errorf("secure config file: %w", err)
	}
	return nil
}

func EnsureRuntimeDirs(cfg *Config) error {
	for _, dir := range []string{
		cfg.Paths.StateDir,
		cfg.Paths.LogDir,
		cfg.Paths.SecretsDir,
		cfg.Paths.AgentDir,
		cfg.Paths.EnrollmentDir,
		cfg.Paths.JoinCodesDir,
		cfg.Paths.TrustedNodesDir,
		cfg.Paths.AuditDir,
	} {
		if dir == "" {
			return errors.New("runtime directory path is empty")
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create runtime directory %q: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure runtime directory %q: %w", dir, err)
		}
	}
	return nil
}

func RoleStateDir(cfg *Config) string {
	if cfg == nil || cfg.Node.Role == "" {
		return ""
	}
	return filepath.Join(cfg.Paths.StateDir, cfg.Node.Role)
}

func MarkInitialized(cfg *Config, role string, now time.Time) (bool, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if !ValidRole(role) {
		return false, fmt.Errorf("unsupported role %q: expected master or worker", role)
	}
	if cfg.Node.Role != "" {
		if cfg.Node.Role == role {
			return false, nil
		}
		return false, fmt.Errorf("node is already initialized as %s; refusing to change role", cfg.Node.Role)
	}
	cfg.Node.Role = role
	cfg.Node.InitializedAt = now.UTC()
	if cfg.Node.ID == "" {
		id, err := NewNodeID("node")
		if err != nil {
			return false, err
		}
		cfg.Node.ID = id
	}
	if role == RoleMaster && cfg.Cluster.ID == "" {
		clusterID, err := NewNodeID("cluster")
		if err != nil {
			return false, err
		}
		cfg.Cluster.ID = clusterID
		cfg.Cluster.Name = "tailedbox"
	}
	return true, nil
}

func ValidRole(role string) bool {
	return role == RoleMaster || role == RoleWorker
}

func NewNodeID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate node id: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return prefix + "_" + strings.ToLower(encoded), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func defaultConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "tailedbox"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "tailedbox"), nil
}

func defaultStateDir() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "tailedbox"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "tailedbox"), nil
}
