package nodeinit

import (
	"time"

	"github.com/tailedbox/tailedbox/internal/agent"
	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/identity"
	"github.com/tailedbox/tailedbox/internal/node"
)

type Result struct {
	IdentityCreated       bool
	PublicIdentityChanged bool
	NodeMetadataChanged   bool
	AgentConfigChanged    bool
	RoleDir               string
	PrivateKeyFile        string
	PublicIdentityFile    string
	NodeMetadataFile      string
	AgentConfigFile       string
	IdentityFingerprint   string
}

func Ensure(cfg *config.Config, now time.Time) (Result, error) {
	identityResult, err := identity.Ensure(cfg, now)
	if err != nil {
		return Result{}, err
	}
	metadataResult, err := node.EnsureMetadata(cfg, now)
	if err != nil {
		return Result{}, err
	}
	agentResult, err := agent.EnsureConfig(cfg, now)
	if err != nil {
		return Result{}, err
	}
	return Result{
		IdentityCreated:       identityResult.Created,
		PublicIdentityChanged: identityResult.PublicIdentityChanged,
		NodeMetadataChanged:   metadataResult.Changed,
		AgentConfigChanged:    agentResult.Changed,
		RoleDir:               metadataResult.RoleDir,
		PrivateKeyFile:        identityResult.PrivateKeyFile,
		PublicIdentityFile:    identityResult.PublicIdentityFile,
		NodeMetadataFile:      metadataResult.Path,
		AgentConfigFile:       agentResult.Path,
		IdentityFingerprint:   identityResult.PublicIdentity.PublicKeyFingerprint,
	}, nil
}
