package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
)

const PeerExportVersion = 1

type PeerExport struct {
	Version    int                         `json:"version"`
	NodeID     secureidentity.NodeID       `json:"node_id"`
	NetworkID  secureidentity.NetworkID    `json:"network_id"`
	Role       secureidentity.Role         `json:"role"`
	PublicKeys secureidentity.PublicKeySet `json:"public_keys"`
}

func (e PeerExport) Validate() error {
	if e.Version != PeerExportVersion {
		return fmt.Errorf("unsupported peer export version %d", e.Version)
	}
	if err := e.NodeID.Validate(); err != nil {
		return err
	}
	if err := e.NetworkID.Validate(); err != nil {
		return err
	}
	if err := e.Role.Validate(); err != nil {
		return err
	}
	return e.PublicKeys.Validate()
}

func DecodePeerExport(data []byte) (PeerExport, error) {
	var exported PeerExport
	if err := json.Unmarshal(data, &exported); err != nil {
		return PeerExport{}, err
	}
	if err := exported.Validate(); err != nil {
		return PeerExport{}, err
	}
	return exported, nil
}

func ExportPeer(ctx context.Context, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	local, err := secureidentity.LoadIdentity(env.paths)
	if err != nil {
		return Result{}, err
	}
	exported := PeerExport{
		Version:    PeerExportVersion,
		NodeID:     local.NodeID,
		NetworkID:  local.NetworkID,
		Role:       local.Role,
		PublicKeys: local.PublicKeys,
	}
	data, err := json.MarshalIndent(exported, "", "  ")
	if err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "peer", "export"),
		RawOutput:     string(append(data, '\n')),
	}, nil
}

func AddPeer(ctx context.Context, exported PeerExport, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	if err := exported.Validate(); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	local, err := secureidentity.LoadIdentity(env.paths)
	if err != nil {
		return Result{}, err
	}
	if exported.NetworkID != local.NetworkID {
		return Result{}, fmt.Errorf("peer network %q does not match local network %q", exported.NetworkID, local.NetworkID)
	}
	record := peer.Record{
		NodeID:     exported.NodeID,
		NetworkID:  exported.NetworkID,
		Role:       exported.Role,
		PublicKeys: exported.PublicKeys,
		Status:     peer.StatusActive,
		AddedAt:    env.nowUTC(),
	}
	if err := peer.NewStore(env.paths).Add(record); err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "peer", "add", "--file", "<peer.json>"),
		Message:       "peer added",
		Fields: map[string]string{
			"node_id": exported.NodeID.String(),
			"role":    exported.Role.String(),
		},
	}, nil
}

func ListPeers(ctx context.Context, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	records, err := peer.NewStore(env.paths).List()
	if err != nil {
		return Result{}, err
	}
	items := make([]map[string]string, 0, len(records))
	for _, record := range records {
		items = append(items, map[string]string{
			"node_id": record.NodeID.String(),
			"role":    record.Role.String(),
			"status":  string(record.Status),
		})
	}
	return Result{
		EquivalentCLI: Command("infra", "peer", "list"),
		Message:       "peers loaded",
		Fields: map[string]string{
			"count": strconv.Itoa(len(records)),
		},
		Items: items,
	}, nil
}
