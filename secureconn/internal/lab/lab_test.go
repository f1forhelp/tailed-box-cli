package lab

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInitPairAndLoadTrust(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	masterDir := t.TempDir()
	workerDir := t.TempDir()
	master, created, err := Init(InitOptions{
		StateDir:  masterDir,
		Role:      RoleMaster,
		NodeID:    "lab_master",
		ClusterID: "cluster_lab",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("init master: %v", err)
	}
	if !created {
		t.Fatal("expected master creation")
	}
	worker, _, err := Init(InitOptions{
		StateDir:  workerDir,
		Role:      RoleWorker,
		NodeID:    "lab_worker",
		ClusterID: "cluster_lab",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("init worker: %v", err)
	}
	if err := Pair(PairOptions{
		MasterStateDir: masterDir,
		WorkerStateDir: workerDir,
		MasterEndpoint: "127.0.0.1:41677",
		TrustBothWays:  true,
		Now:            now,
	}); err != nil {
		t.Fatalf("pair: %v", err)
	}
	workerTrust, err := LoadTrust(workerDir)
	if err != nil {
		t.Fatalf("load worker trust: %v", err)
	}
	if got := workerTrust.Peers[master.NodeID]; got.LastEndpoint != "127.0.0.1:41677" {
		t.Fatalf("unexpected worker trust: %#v", got)
	}
	masterTrust, err := LoadTrust(masterDir)
	if err != nil {
		t.Fatalf("load master trust: %v", err)
	}
	if got := masterTrust.Peers[worker.NodeID]; got.NodeID != worker.NodeID {
		t.Fatalf("unexpected master trust: %#v", got)
	}
	if err := RemoveTrust(masterDir, worker.NodeID); err != nil {
		t.Fatalf("remove trust: %v", err)
	}
	masterTrust, err = LoadTrust(masterDir)
	if err != nil {
		t.Fatalf("reload master trust: %v", err)
	}
	if _, ok := masterTrust.Peers[worker.NodeID]; ok {
		t.Fatalf("worker trust was not revoked: %#v", masterTrust.Peers[worker.NodeID])
	}
}

func TestInviteJoinPersistsTrustAndEndpoints(t *testing.T) {
	now := time.Now().UTC()
	masterDir := t.TempDir()
	workerDir := t.TempDir()
	master, _, err := Init(InitOptions{
		StateDir:  masterDir,
		Role:      RoleMaster,
		NodeID:    "lab_master",
		ClusterID: "cluster_lab",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("init master: %v", err)
	}
	worker, _, err := Init(InitOptions{
		StateDir:  workerDir,
		Role:      RoleWorker,
		NodeID:    "lab_worker",
		ClusterID: "cluster_lab",
		Now:       now,
	})
	if err != nil {
		t.Fatalf("init worker: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	masterTransport, _, err := Start(ctx, masterDir, "127.0.0.1", 0)
	if err != nil {
		t.Fatalf("start master: %v", err)
	}
	defer masterTransport.Close()

	publicEndpoint := masterTransport.BoundEndpoint()
	vpcEndpoint := Endpoint("10.0.1.5", masterTransport.BoundUDPPort())
	invite, err := CreateInvite(InviteOptions{
		StateDir:       masterDir,
		Role:           RoleWorker,
		TTL:            DefaultInviteTTL,
		PublicEndpoint: publicEndpoint,
		VPCEndpoint:    vpcEndpoint,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	codeParts := strings.Split(invite.Code, ".")
	if len(codeParts) != 3 {
		t.Fatalf("unexpected invite code: %q", invite.Code)
	}
	inviteBytes, err := os.ReadFile(inviteFile(masterDir, invite.Invite.ID))
	if err != nil {
		t.Fatalf("read invite file: %v", err)
	}
	if strings.Contains(string(inviteBytes), codeParts[2]) {
		t.Fatalf("invite file persisted raw secret")
	}

	joined, err := Join(ctx, JoinOptions{
		StateDir:       workerDir,
		Code:           invite.Code,
		MasterEndpoint: publicEndpoint,
		Timeout:        5 * time.Second,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if joined.MasterNodeID != master.NodeID {
		t.Fatalf("joined master = %s, want %s", joined.MasterNodeID, master.NodeID)
	}
	workerTrust, err := LoadTrust(workerDir)
	if err != nil {
		t.Fatalf("load worker trust: %v", err)
	}
	trustedMaster := workerTrust.Peers[master.NodeID]
	if trustedMaster.LastEndpoint != publicEndpoint || trustedMaster.Endpoints.Public != publicEndpoint || trustedMaster.Endpoints.VPC != vpcEndpoint {
		t.Fatalf("unexpected trusted master endpoints: %#v", trustedMaster)
	}
	masterTrust, err := LoadTrust(masterDir)
	if err != nil {
		t.Fatalf("load master trust: %v", err)
	}
	if got := masterTrust.Peers[worker.NodeID]; got.NodeID != worker.NodeID || got.LastEndpoint == "" {
		t.Fatalf("unexpected trusted worker: %#v", got)
	}
	invites, err := ListInvites(masterDir)
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}
	if len(invites) != 1 || invites[0].UsedByNodeID != worker.NodeID || invites[0].UsedAt.IsZero() {
		t.Fatalf("invite was not consumed: %#v", invites)
	}
	if _, err := Join(ctx, JoinOptions{StateDir: workerDir, Code: invite.Code, MasterEndpoint: publicEndpoint, Timeout: 5 * time.Second, Now: now}); err == nil {
		t.Fatalf("expected reused invite to be rejected")
	}
	if _, err := Ping(ctx, workerDir, master.NodeID, publicEndpoint); err != nil {
		t.Fatalf("ping after join: %v", err)
	}
}

func TestInviteCodePinsMasterIdentityAndFreshNonce(t *testing.T) {
	now := time.Now().UTC()
	masterDir := t.TempDir()
	workerDir := t.TempDir()
	rogueDir := t.TempDir()
	master, _, err := Init(InitOptions{StateDir: masterDir, Role: RoleMaster, NodeID: "real_master", ClusterID: "cluster_lab", Now: now})
	if err != nil {
		t.Fatalf("init master: %v", err)
	}
	worker, _, err := Init(InitOptions{StateDir: workerDir, Role: RoleWorker, NodeID: "worker_1", ClusterID: "cluster_lab", Now: now})
	if err != nil {
		t.Fatalf("init worker: %v", err)
	}
	rogue, _, err := Init(InitOptions{StateDir: rogueDir, Role: RoleMaster, NodeID: "rogue_master", ClusterID: "cluster_lab", Now: now})
	if err != nil {
		t.Fatalf("init rogue: %v", err)
	}
	invite, err := CreateInvite(InviteOptions{StateDir: masterDir, Role: RoleWorker, TTL: DefaultInviteTTL, PublicEndpoint: "127.0.0.1:24177", Now: now})
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	parsed, err := parseInviteCode(invite.Code)
	if err != nil {
		t.Fatalf("parse invite code: %v", err)
	}
	if parsed.Payload.IssuerNodeID != master.NodeID || parsed.Payload.IssuerIdentityFingerprint != master.PublicIdentity.PublicKeyFingerprint {
		t.Fatalf("invite did not pin master identity: %#v", parsed.Payload)
	}

	request := enrollmentRequest{
		Version:             enrollmentVersion,
		CodeID:              parsed.Payload.ID,
		ClusterID:           parsed.Payload.ClusterID,
		NodeID:              worker.NodeID,
		Role:                worker.Role,
		IdentityFingerprint: worker.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:      worker.PublicIdentity,
		ClientNonce:         "fresh-client-nonce",
		SentAt:              now,
	}
	rogueChallenge := enrollmentChallenge{
		Version:             enrollmentVersion,
		CodeID:              parsed.Payload.ID,
		ChallengeID:         "challenge_1",
		ClusterID:           parsed.Payload.ClusterID,
		NodeID:              rogue.NodeID,
		Role:                rogue.Role,
		IdentityFingerprint: rogue.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:      rogue.PublicIdentity,
		ClientNonce:         request.ClientNonce,
		ChallengeNonce:      "challenge-nonce",
		SentAt:              now,
		Signature:           "signature",
	}
	if err := validateChallengeAgainstInvite(rogueChallenge, parsed.Payload, request); err == nil {
		t.Fatal("expected rogue master challenge to be rejected")
	}
	replayedChallenge := rogueChallenge
	replayedChallenge.NodeID = master.NodeID
	replayedChallenge.Role = master.Role
	replayedChallenge.IdentityFingerprint = master.PublicIdentity.PublicKeyFingerprint
	replayedChallenge.PublicIdentity = master.PublicIdentity
	replayedChallenge.ClientNonce = "old-client-nonce"
	if err := validateChallengeAgainstInvite(replayedChallenge, parsed.Payload, request); err == nil {
		t.Fatal("expected replayed challenge nonce to be rejected")
	}
	validChallenge := replayedChallenge
	validChallenge.ClientNonce = request.ClientNonce
	accept := enrollmentAccept{
		Version:                 enrollmentVersion,
		CodeID:                  parsed.Payload.ID,
		ChallengeID:             validChallenge.ChallengeID,
		ClusterID:               parsed.Payload.ClusterID,
		NodeID:                  master.NodeID,
		Role:                    master.Role,
		IdentityFingerprint:     master.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:          master.PublicIdentity,
		PeerNodeID:              worker.NodeID,
		PeerIdentityFingerprint: worker.PublicIdentity.PublicKeyFingerprint,
		ClientNonce:             "old-client-nonce",
		ChallengeNonce:          validChallenge.ChallengeNonce,
		SentAt:                  now,
		Signature:               "signature",
	}
	if err := validateAcceptAgainstInvite(accept, parsed.Payload, request, validChallenge); err == nil {
		t.Fatal("expected replayed accept nonce to be rejected")
	}
}
