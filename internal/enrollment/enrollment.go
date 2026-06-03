package enrollment

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tailedbox/tailedbox/internal/audit"
	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/identity"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	DefaultJoinCodeTTL     = 15 * time.Minute
	MaxJoinCodeTTL         = 24 * time.Hour
	DefaultReconnectWindow = 24 * time.Hour
	MaxReconnectWindow     = 30 * 24 * time.Hour
)

type CreateJoinCodeOptions struct {
	AllowedRole     string
	TTL             time.Duration
	ReconnectWindow time.Duration
	Now             time.Time
}

type CreateJoinCodeResult struct {
	JoinCode            string    `json:"join_code"`
	CodeID              string    `json:"code_id"`
	AllowedRole         string    `json:"allowed_role"`
	ClusterID           string    `json:"cluster_id"`
	ClusterName         string    `json:"cluster_name,omitempty"`
	IssuerNodeID        string    `json:"issuer_node_id"`
	IssuerFingerprint   string    `json:"issuer_fingerprint"`
	ExpiresAt           time.Time `json:"expires_at"`
	ReconnectWindow     string    `json:"reconnect_window"`
	LocalMasterStateDir string    `json:"local_master_state_dir"`
}

type JoinOptions struct {
	ExpectedRole   string
	RawCode        string
	MasterStateDir string
	Now            time.Time
}

type JoinResult struct {
	NodeID                    string    `json:"node_id"`
	Role                      string    `json:"role"`
	ClusterID                 string    `json:"cluster_id"`
	ClusterName               string    `json:"cluster_name,omitempty"`
	MasterNodeID              string    `json:"master_node_id"`
	MasterIdentityFingerprint string    `json:"master_identity_fingerprint"`
	JoinCodeID                string    `json:"join_code_id"`
	ReconnectLeaseExpiresAt   time.Time `json:"reconnect_lease_expires_at"`
}

func CreateJoinCode(cfg *config.Config, opts CreateJoinCodeOptions) (CreateJoinCodeResult, error) {
	if cfg == nil {
		return CreateJoinCodeResult{}, errors.New("config is nil")
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if cfg.Node.Role != config.RoleMaster {
		return CreateJoinCodeResult{}, errors.New("only initialized master nodes can create join codes")
	}
	if cfg.Node.ID == "" || cfg.Cluster.ID == "" {
		return CreateJoinCodeResult{}, errors.New("master node is not fully initialized")
	}
	if cfg.Node.Identity.PublicKeyFingerprint == "" || !secrets.Exists(cfg.Paths.IdentityPrivateKeyFile) {
		return CreateJoinCodeResult{}, errors.New("master node identity is not initialized")
	}
	if !config.ValidRole(opts.AllowedRole) {
		return CreateJoinCodeResult{}, fmt.Errorf("unsupported join role %q", opts.AllowedRole)
	}
	ttl := opts.TTL
	if ttl == 0 {
		ttl = DefaultJoinCodeTTL
	}
	if ttl <= 0 || ttl > MaxJoinCodeTTL {
		return CreateJoinCodeResult{}, fmt.Errorf("join code ttl must be greater than 0 and no more than %s", MaxJoinCodeTTL)
	}
	reconnectWindow := opts.ReconnectWindow
	if reconnectWindow == 0 {
		reconnectWindow = DefaultReconnectWindow
	}
	if reconnectWindow <= 0 || reconnectWindow > MaxReconnectWindow {
		return CreateJoinCodeResult{}, fmt.Errorf("reconnect window must be greater than 0 and no more than %s", MaxReconnectWindow)
	}
	codeID, err := NewCodeID()
	if err != nil {
		return CreateJoinCodeResult{}, err
	}
	payload := JoinCodePayload{
		Version:           1,
		CodeID:            codeID,
		AllowedRole:       opts.AllowedRole,
		ClusterID:         cfg.Cluster.ID,
		ClusterName:       cfg.Cluster.Name,
		IssuerNodeID:      cfg.Node.ID,
		IssuerFingerprint: cfg.Node.Identity.PublicKeyFingerprint,
		IssuedAt:          now,
		ExpiresAt:         now.Add(ttl),
	}
	code, hash, err := NewJoinCode(payload)
	if err != nil {
		return CreateJoinCodeResult{}, err
	}
	record := JoinCodeRecord{
		Version:             1,
		ID:                  codeID,
		AllowedRole:         opts.AllowedRole,
		ClusterID:           cfg.Cluster.ID,
		ClusterName:         cfg.Cluster.Name,
		IssuerNodeID:        cfg.Node.ID,
		IssuerFingerprint:   cfg.Node.Identity.PublicKeyFingerprint,
		SecretHash:          hash,
		SecretHashAlgorithm: HashSHA256,
		State:               JoinCodeStateActive,
		CreatedAt:           now,
		ExpiresAt:           payload.ExpiresAt,
		ReconnectWindow:     reconnectWindow.String(),
	}
	if err := WriteJoinCodeRecord(cfg.Paths, record); err != nil {
		return CreateJoinCodeResult{}, err
	}
	_ = audit.Append(cfg.Paths.AuditLogFile, audit.Event{
		Time:        now,
		Action:      audit.ActionJoinCodeCreated,
		ActorNodeID: cfg.Node.ID,
		Role:        opts.AllowedRole,
		JoinCodeID:  codeID,
		Result:      "ok",
	})
	return CreateJoinCodeResult{
		JoinCode:            code,
		CodeID:              codeID,
		AllowedRole:         opts.AllowedRole,
		ClusterID:           cfg.Cluster.ID,
		ClusterName:         cfg.Cluster.Name,
		IssuerNodeID:        cfg.Node.ID,
		IssuerFingerprint:   cfg.Node.Identity.PublicKeyFingerprint,
		ExpiresAt:           payload.ExpiresAt,
		ReconnectWindow:     reconnectWindow.String(),
		LocalMasterStateDir: cfg.Paths.StateDir,
	}, nil
}

func Join(cfg *config.Config, opts JoinOptions) (JoinResult, error) {
	if cfg == nil {
		return JoinResult{}, errors.New("config is nil")
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !config.ValidRole(opts.ExpectedRole) {
		return JoinResult{}, fmt.Errorf("unsupported join role %q", opts.ExpectedRole)
	}
	if cfg.Node.Role != opts.ExpectedRole {
		return JoinResult{}, fmt.Errorf("this node is initialized as %s; cannot join as %s", roleOrUninitialized(cfg.Node.Role), opts.ExpectedRole)
	}
	if cfg.Node.ID == "" {
		return JoinResult{}, errors.New("node id is missing; run tailedbox init first")
	}
	publicIdentity, err := identity.LoadPublicIdentity(cfg.Paths.IdentityPublicKeyFile)
	if err != nil {
		return JoinResult{}, fmt.Errorf("load local public identity: %w", err)
	}
	if publicIdentity.NodeID != cfg.Node.ID {
		return JoinResult{}, fmt.Errorf("local public identity belongs to node %s, expected %s", publicIdentity.NodeID, cfg.Node.ID)
	}
	parsed, err := ParseJoinCode(opts.RawCode)
	if err != nil {
		_ = audit.Append(cfg.Paths.AuditLogFile, audit.Event{
			Time:        now,
			Action:      audit.ActionJoinFailed,
			ActorNodeID: cfg.Node.ID,
			Role:        opts.ExpectedRole,
			Result:      "rejected",
			Reason:      "invalid_join_code",
		})
		return JoinResult{}, err
	}
	if opts.MasterStateDir == "" {
		return JoinResult{}, errors.New("local enrollment requires --master-state-dir until mesh transport is implemented")
	}
	masterPaths, err := pathsForStateDir(opts.MasterStateDir)
	if err != nil {
		return JoinResult{}, err
	}
	appendAttempt(masterPaths, now, cfg.Node.ID, opts.ExpectedRole, parsed.Payload.CodeID)

	result, err := joinParsed(cfg, publicIdentity, parsed, masterPaths, opts.ExpectedRole, now)
	if err != nil {
		_ = audit.Append(masterPaths.AuditLogFile, audit.Event{
			Time:        now,
			Action:      audit.ActionJoinFailed,
			ActorNodeID: cfg.Node.ID,
			Role:        opts.ExpectedRole,
			JoinCodeID:  parsed.Payload.CodeID,
			Result:      "rejected",
			Reason:      err.Error(),
		})
		return JoinResult{}, err
	}
	_ = audit.Append(masterPaths.AuditLogFile, audit.Event{
		Time:         now,
		Action:       audit.ActionJoinSucceeded,
		ActorNodeID:  cfg.Node.ID,
		TargetNodeID: cfg.Node.ID,
		Role:         opts.ExpectedRole,
		JoinCodeID:   parsed.Payload.CodeID,
		Result:       "ok",
	})
	_ = audit.Append(cfg.Paths.AuditLogFile, audit.Event{
		Time:        now,
		Action:      audit.ActionJoinSucceeded,
		ActorNodeID: cfg.Node.ID,
		Role:        opts.ExpectedRole,
		JoinCodeID:  parsed.Payload.CodeID,
		Result:      "ok",
	})
	return result, nil
}

func joinParsed(cfg *config.Config, publicIdentity identity.PublicIdentity, parsed ParsedJoinCode, masterPaths config.Paths, expectedRole string, now time.Time) (JoinResult, error) {
	payload := parsed.Payload
	if payload.AllowedRole != expectedRole {
		return JoinResult{}, fmt.Errorf("join code is scoped to %s nodes", payload.AllowedRole)
	}
	if existing, err := ReadJoinedCluster(cfg.Paths.JoinedClusterFile); err == nil {
		return JoinResult{}, fmt.Errorf("node is already joined to cluster %s with join code %s", existing.ClusterID, existing.JoinCodeID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return JoinResult{}, err
	}
	record, err := ReadJoinCodeRecord(masterPaths, payload.CodeID)
	if err != nil {
		return JoinResult{}, fmt.Errorf("join code is not recognized by the master state")
	}
	if err := validateRecord(record, parsed, expectedRole, now); err != nil {
		if err == errExpiredJoinCode {
			markExpired(masterPaths, record)
		}
		return JoinResult{}, err
	}
	if TrustedNodeExists(masterPaths, cfg.Node.ID) {
		return JoinResult{}, fmt.Errorf("node identity %s is already trusted by the master cluster", cfg.Node.ID)
	}
	reconnectWindow, err := time.ParseDuration(record.ReconnectWindow)
	if err != nil || reconnectWindow <= 0 {
		reconnectWindow = DefaultReconnectWindow
	}
	leaseExpires := now.Add(reconnectWindow).UTC()
	trusted := TrustedNode{
		Version:                 1,
		NodeID:                  cfg.Node.ID,
		Role:                    expectedRole,
		IdentityAlgorithm:       publicIdentity.Algorithm,
		IdentityFingerprint:     publicIdentity.PublicKeyFingerprint,
		PublicKey:               publicIdentity.PublicKey,
		ClusterID:               record.ClusterID,
		ClusterName:             record.ClusterName,
		JoinCodeID:              record.ID,
		TrustState:              TrustStateTrusted,
		JoinedAt:                now,
		ReconnectLeaseExpiresAt: leaseExpires,
	}
	if err := WriteTrustedNode(masterPaths, trusted); err != nil {
		return JoinResult{}, err
	}
	record.State = JoinCodeStateUsed
	usedAt := now.UTC()
	record.UsedAt = &usedAt
	record.UsedByNodeID = cfg.Node.ID
	record.UsedByFingerprint = publicIdentity.PublicKeyFingerprint
	record.ReconnectLeaseExpiresAt = &leaseExpires
	if err := WriteJoinCodeRecord(masterPaths, record); err != nil {
		return JoinResult{}, err
	}
	joined := JoinedCluster{
		Version:                   1,
		NodeID:                    cfg.Node.ID,
		Role:                      expectedRole,
		ClusterID:                 record.ClusterID,
		ClusterName:               record.ClusterName,
		MasterNodeID:              record.IssuerNodeID,
		MasterIdentityFingerprint: record.IssuerFingerprint,
		JoinCodeID:                record.ID,
		JoinedAt:                  now,
		ReconnectLeaseExpiresAt:   leaseExpires,
	}
	if err := WriteJoinedCluster(cfg.Paths.JoinedClusterFile, joined); err != nil {
		return JoinResult{}, err
	}
	cfg.Cluster.ID = record.ClusterID
	cfg.Cluster.Name = record.ClusterName
	return JoinResult{
		NodeID:                    cfg.Node.ID,
		Role:                      expectedRole,
		ClusterID:                 record.ClusterID,
		ClusterName:               record.ClusterName,
		MasterNodeID:              record.IssuerNodeID,
		MasterIdentityFingerprint: record.IssuerFingerprint,
		JoinCodeID:                record.ID,
		ReconnectLeaseExpiresAt:   leaseExpires,
	}, nil
}

var errExpiredJoinCode = errors.New("join code has expired")

func validateRecord(record JoinCodeRecord, parsed ParsedJoinCode, expectedRole string, now time.Time) error {
	payload := parsed.Payload
	if record.ID != payload.CodeID || record.ClusterID != payload.ClusterID || record.IssuerNodeID != payload.IssuerNodeID {
		return errors.New("join code does not match the master record")
	}
	if record.AllowedRole != expectedRole || record.AllowedRole != payload.AllowedRole {
		return fmt.Errorf("join code is scoped to %s nodes", record.AllowedRole)
	}
	if record.SecretHashAlgorithm != HashSHA256 || record.SecretHash != HashForParsed(parsed) {
		return errors.New("join code secret was rejected")
	}
	if record.State == JoinCodeStateUsed || record.UsedAt != nil {
		return errors.New("join code has already been used")
	}
	if record.State == JoinCodeStateExpired || !now.Before(record.ExpiresAt) {
		return errExpiredJoinCode
	}
	return nil
}

func markExpired(paths config.Paths, record JoinCodeRecord) {
	record.State = JoinCodeStateExpired
	_ = WriteJoinCodeRecord(paths, record)
}

func appendAttempt(paths config.Paths, now time.Time, nodeID, role, codeID string) {
	_ = audit.Append(paths.AuditLogFile, audit.Event{
		Time:        now,
		Action:      audit.ActionJoinAttempt,
		ActorNodeID: nodeID,
		Role:        role,
		JoinCodeID:  codeID,
		Result:      "attempt",
	})
}

func pathsForStateDir(stateDir string) (config.Paths, error) {
	paths, err := config.ResolvePaths(config.LoadOptions{StateDir: stateDir})
	if err != nil {
		return config.Paths{}, err
	}
	return paths, nil
}

func roleOrUninitialized(role string) string {
	if role == "" {
		return "uninitialized"
	}
	return role
}
