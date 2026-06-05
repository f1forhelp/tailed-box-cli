package lab

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tailedbox/link/identity"
	"github.com/tailedbox/link/internal/private"
	"github.com/tailedbox/link/protocol"
	"github.com/tailedbox/link/transport"
)

const (
	DefaultInviteTTL       = 15 * time.Minute
	enrollmentVersion      = 1
	inviteCodePrefix       = "scj1"
	enrollmentChallengeTTL = 2 * time.Minute
	enrollmentSkew         = 5 * time.Minute
	enrollmentMaxPacket    = protocol.HeaderLen + protocol.MaxPayloadSize
)

type InviteOptions struct {
	StateDir       string
	Role           string
	TTL            time.Duration
	PublicEndpoint string
	VPCEndpoint    string
	Now            time.Time
}

type Invite struct {
	Version                   int         `json:"version"`
	ID                        string      `json:"id"`
	Role                      string      `json:"role"`
	ClusterID                 string      `json:"cluster_id"`
	IssuerNodeID              string      `json:"issuer_node_id"`
	IssuerIdentityFingerprint string      `json:"issuer_identity_fingerprint"`
	SecretHash                string      `json:"secret_hash"`
	Endpoints                 EndpointSet `json:"endpoints,omitempty"`
	CreatedAt                 time.Time   `json:"created_at"`
	ExpiresAt                 time.Time   `json:"expires_at"`
	UsedAt                    time.Time   `json:"used_at,omitempty"`
	UsedByNodeID              string      `json:"used_by_node_id,omitempty"`
	UsedByEndpoint            string      `json:"used_by_endpoint,omitempty"`
}

type InviteResult struct {
	Code   string `json:"code"`
	Invite Invite `json:"invite"`
}

type JoinOptions struct {
	StateDir       string
	Code           string
	MasterEndpoint string
	PublicEndpoint string
	VPCEndpoint    string
	Timeout        time.Duration
	Now            time.Time
}

type JoinResult struct {
	MasterNodeID   string      `json:"master_node_id"`
	MasterEndpoint string      `json:"master_endpoint"`
	Endpoints      EndpointSet `json:"endpoints,omitempty"`
	TrustedAt      time.Time   `json:"trusted_at"`
}

type parsedInviteCode struct {
	Payload    inviteCodePayload
	SecretHash []byte
}

type inviteCodePayload struct {
	Version                   int       `json:"version"`
	ID                        string    `json:"id"`
	Role                      string    `json:"role"`
	ClusterID                 string    `json:"cluster_id"`
	IssuerNodeID              string    `json:"issuer_node_id"`
	IssuerIdentityFingerprint string    `json:"issuer_identity_fingerprint"`
	ExpiresAt                 time.Time `json:"expires_at"`
}

type EnrollmentHandler struct {
	StateDir string
	Node     Node

	mu      sync.Mutex
	pending map[string]pendingEnrollment
}

type pendingEnrollment struct {
	Request        enrollmentRequest
	Challenge      enrollmentChallenge
	RemoteEndpoint string
	ExpiresAt      time.Time
}

type enrollmentRequest struct {
	Version             int                     `json:"version"`
	CodeID              string                  `json:"code_id"`
	ClusterID           string                  `json:"cluster_id"`
	NodeID              string                  `json:"node_id"`
	Role                string                  `json:"role"`
	IdentityFingerprint string                  `json:"identity_fingerprint"`
	PublicIdentity      identity.PublicIdentity `json:"public_identity"`
	Endpoints           EndpointSet             `json:"endpoints,omitempty"`
	ClientNonce         string                  `json:"client_nonce"`
	SentAt              time.Time               `json:"sent_at"`
}

type enrollmentChallenge struct {
	Version             int                     `json:"version"`
	CodeID              string                  `json:"code_id"`
	ChallengeID         string                  `json:"challenge_id"`
	ClusterID           string                  `json:"cluster_id"`
	NodeID              string                  `json:"node_id"`
	Role                string                  `json:"role"`
	IdentityFingerprint string                  `json:"identity_fingerprint"`
	PublicIdentity      identity.PublicIdentity `json:"public_identity"`
	Endpoints           EndpointSet             `json:"endpoints,omitempty"`
	ClientNonce         string                  `json:"client_nonce"`
	ChallengeNonce      string                  `json:"challenge_nonce"`
	SentAt              time.Time               `json:"sent_at"`
	Signature           string                  `json:"signature"`
}

type enrollmentProof struct {
	Version             int       `json:"version"`
	CodeID              string    `json:"code_id"`
	ChallengeID         string    `json:"challenge_id"`
	NodeID              string    `json:"node_id"`
	Role                string    `json:"role"`
	IdentityFingerprint string    `json:"identity_fingerprint"`
	SentAt              time.Time `json:"sent_at"`
	Proof               string    `json:"proof"`
}

type enrollmentAccept struct {
	Version                 int                     `json:"version"`
	CodeID                  string                  `json:"code_id"`
	ChallengeID             string                  `json:"challenge_id"`
	ClusterID               string                  `json:"cluster_id"`
	NodeID                  string                  `json:"node_id"`
	Role                    string                  `json:"role"`
	IdentityFingerprint     string                  `json:"identity_fingerprint"`
	PublicIdentity          identity.PublicIdentity `json:"public_identity"`
	Endpoints               EndpointSet             `json:"endpoints,omitempty"`
	PeerNodeID              string                  `json:"peer_node_id"`
	PeerIdentityFingerprint string                  `json:"peer_identity_fingerprint"`
	ClientNonce             string                  `json:"client_nonce"`
	ChallengeNonce          string                  `json:"challenge_nonce"`
	SentAt                  time.Time               `json:"sent_at"`
	Signature               string                  `json:"signature"`
}

type enrollmentReject struct {
	Version     int       `json:"version"`
	CodeID      string    `json:"code_id,omitempty"`
	ChallengeID string    `json:"challenge_id,omitempty"`
	Reason      string    `json:"reason"`
	SentAt      time.Time `json:"sent_at"`
}

type enrollmentProofTranscript struct {
	Version                   int         `json:"version"`
	CodeID                    string      `json:"code_id"`
	ChallengeID               string      `json:"challenge_id"`
	ClusterID                 string      `json:"cluster_id"`
	WorkerNodeID              string      `json:"worker_node_id"`
	WorkerRole                string      `json:"worker_role"`
	WorkerIdentityFingerprint string      `json:"worker_identity_fingerprint"`
	WorkerPublicKey           string      `json:"worker_public_key"`
	WorkerEndpoints           EndpointSet `json:"worker_endpoints,omitempty"`
	MasterNodeID              string      `json:"master_node_id"`
	MasterRole                string      `json:"master_role"`
	MasterIdentityFingerprint string      `json:"master_identity_fingerprint"`
	MasterPublicKey           string      `json:"master_public_key"`
	MasterEndpoints           EndpointSet `json:"master_endpoints,omitempty"`
	ClientNonce               string      `json:"client_nonce"`
	ChallengeNonce            string      `json:"challenge_nonce"`
}

func CreateInvite(opts InviteOptions) (InviteResult, error) {
	if opts.StateDir == "" {
		return InviteResult{}, errors.New("state directory is required")
	}
	node, err := LoadNode(opts.StateDir)
	if err != nil {
		return InviteResult{}, err
	}
	if node.Role != RoleMaster {
		return InviteResult{}, fmt.Errorf("invite creation requires a %s node", RoleMaster)
	}
	role := strings.TrimSpace(opts.Role)
	if role == "" {
		role = RoleWorker
	}
	if role != RoleWorker && role != RoleMaster {
		return InviteResult{}, fmt.Errorf("invite role must be %q or %q", RoleWorker, RoleMaster)
	}
	if err := validateEndpointSet(EndpointSet{Public: opts.PublicEndpoint, VPC: opts.VPCEndpoint}); err != nil {
		return InviteResult{}, err
	}
	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = DefaultInviteTTL
	}
	id, err := randomToken(12)
	if err != nil {
		return InviteResult{}, err
	}
	secret, err := randomToken(32)
	if err != nil {
		return InviteResult{}, err
	}
	secretHash := hashInviteSecret(secret)
	invite := Invite{
		Version:                   1,
		ID:                        id,
		Role:                      role,
		ClusterID:                 node.ClusterID,
		IssuerNodeID:              node.NodeID,
		IssuerIdentityFingerprint: node.PublicIdentity.PublicKeyFingerprint,
		SecretHash:                base64.RawURLEncoding.EncodeToString(secretHash),
		Endpoints: EndpointSet{
			Public: opts.PublicEndpoint,
			VPC:    opts.VPCEndpoint,
		}.Normalized(),
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	if _, err := private.WriteJSONAtomic(inviteFile(opts.StateDir, invite.ID), invite); err != nil {
		return InviteResult{}, err
	}
	codePayload := inviteCodePayload{
		Version:                   enrollmentVersion,
		ID:                        invite.ID,
		Role:                      invite.Role,
		ClusterID:                 invite.ClusterID,
		IssuerNodeID:              invite.IssuerNodeID,
		IssuerIdentityFingerprint: invite.IssuerIdentityFingerprint,
		ExpiresAt:                 invite.ExpiresAt,
	}
	return InviteResult{
		Code:   inviteCodePrefix + "." + base64.RawURLEncoding.EncodeToString(mustEnrollmentJSON(codePayload)) + "." + secret,
		Invite: invite,
	}, nil
}

func ListInvites(stateDir string) ([]Invite, error) {
	dir := inviteDir(stateDir)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read lab invite directory: %w", err)
	}
	invites := make([]Invite, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var invite Invite
		if err := private.ReadJSON(filepath.Join(dir, entry.Name()), &invite); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	sort.Slice(invites, func(i, j int) bool {
		return invites[i].CreatedAt.Before(invites[j].CreatedAt)
	})
	return invites, nil
}

func Join(ctx context.Context, opts JoinOptions) (JoinResult, error) {
	if opts.StateDir == "" {
		return JoinResult{}, errors.New("state directory is required")
	}
	if strings.TrimSpace(opts.MasterEndpoint) == "" {
		return JoinResult{}, errors.New("master endpoint is required")
	}
	if err := validateEndpointSet(EndpointSet{Public: opts.PublicEndpoint, VPC: opts.VPCEndpoint, Last: opts.MasterEndpoint}); err != nil {
		return JoinResult{}, err
	}
	node, err := LoadNode(opts.StateDir)
	if err != nil {
		return JoinResult{}, err
	}
	parsed, err := parseInviteCode(opts.Code)
	if err != nil {
		return JoinResult{}, err
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	remoteAddr, err := net.ResolveUDPAddr("udp", strings.TrimSpace(opts.MasterEndpoint))
	if err != nil {
		return JoinResult{}, fmt.Errorf("resolve master endpoint: %w", err)
	}
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return JoinResult{}, fmt.Errorf("open enrollment UDP socket: %w", err)
	}
	defer conn.Close()

	now := opts.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := validateParsedInviteForJoin(parsed.Payload, node, now); err != nil {
		return JoinResult{}, err
	}
	clientNonce, err := randomToken(24)
	if err != nil {
		return JoinResult{}, err
	}
	request := enrollmentRequest{
		Version:             enrollmentVersion,
		CodeID:              parsed.Payload.ID,
		ClusterID:           node.ClusterID,
		NodeID:              node.NodeID,
		Role:                node.Role,
		IdentityFingerprint: node.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:      node.PublicIdentity,
		Endpoints: EndpointSet{
			Public: opts.PublicEndpoint,
			VPC:    opts.VPCEndpoint,
		}.Normalized(),
		ClientNonce: clientNonce,
		SentAt:      now,
	}
	if err := validateEnrollmentRequest(request, now); err != nil {
		return JoinResult{}, err
	}
	if err := writeEnrollmentPacket(conn, remoteAddr, protocol.Packet{Type: protocol.PacketTypeEnrollRequest, Payload: mustEnrollmentJSON(request)}); err != nil {
		return JoinResult{}, err
	}
	packet, err := readEnrollmentPacket(ctx, conn)
	if err != nil {
		return JoinResult{}, err
	}
	if packet.Type == protocol.PacketTypeEnrollReject {
		return JoinResult{}, decodeEnrollmentReject(packet)
	}
	if packet.Type != protocol.PacketTypeEnrollChallenge {
		return JoinResult{}, fmt.Errorf("expected enrollment challenge, got packet type %d", packet.Type)
	}
	var challenge enrollmentChallenge
	if err := json.Unmarshal(packet.Payload, &challenge); err != nil {
		return JoinResult{}, fmt.Errorf("parse enrollment challenge: %w", err)
	}
	if err := validateEnrollmentChallenge(challenge, time.Now().UTC()); err != nil {
		return JoinResult{}, err
	}
	if challenge.CodeID != request.CodeID {
		return JoinResult{}, errors.New("enrollment challenge code id mismatch")
	}
	if err := validateChallengeAgainstInvite(challenge, parsed.Payload, request); err != nil {
		return JoinResult{}, err
	}
	if err := verifyEnrollmentChallenge(challenge); err != nil {
		return JoinResult{}, err
	}
	proof := enrollmentProof{
		Version:             enrollmentVersion,
		CodeID:              request.CodeID,
		ChallengeID:         challenge.ChallengeID,
		NodeID:              request.NodeID,
		Role:                request.Role,
		IdentityFingerprint: request.IdentityFingerprint,
		SentAt:              time.Now().UTC(),
		Proof:               enrollmentProofValue(parsed.SecretHash, request, challenge),
	}
	if err := writeEnrollmentPacket(conn, remoteAddr, protocol.Packet{Type: protocol.PacketTypeEnrollProof, Payload: mustEnrollmentJSON(proof)}); err != nil {
		return JoinResult{}, err
	}
	packet, err = readEnrollmentPacket(ctx, conn)
	if err != nil {
		return JoinResult{}, err
	}
	if packet.Type == protocol.PacketTypeEnrollReject {
		return JoinResult{}, decodeEnrollmentReject(packet)
	}
	if packet.Type != protocol.PacketTypeEnrollAccept {
		return JoinResult{}, fmt.Errorf("expected enrollment accept, got packet type %d", packet.Type)
	}
	var accept enrollmentAccept
	if err := json.Unmarshal(packet.Payload, &accept); err != nil {
		return JoinResult{}, fmt.Errorf("parse enrollment accept: %w", err)
	}
	if err := validateEnrollmentAccept(accept, time.Now().UTC()); err != nil {
		return JoinResult{}, err
	}
	if accept.CodeID != request.CodeID || accept.ChallengeID != challenge.ChallengeID || accept.PeerNodeID != node.NodeID {
		return JoinResult{}, errors.New("enrollment accept does not match join request")
	}
	if err := validateAcceptAgainstInvite(accept, parsed.Payload, request, challenge); err != nil {
		return JoinResult{}, err
	}
	if err := verifyEnrollmentAccept(accept); err != nil {
		return JoinResult{}, err
	}
	master := Node{
		Version:        1,
		NodeID:         accept.NodeID,
		Role:           accept.Role,
		ClusterID:      accept.ClusterID,
		PublicIdentity: accept.PublicIdentity,
	}
	endpoints := accept.Endpoints.Normalized()
	endpoints.Last = strings.TrimSpace(opts.MasterEndpoint)
	if err := AddTrustWithEndpoints(opts.StateDir, master, endpoints, time.Now().UTC()); err != nil {
		return JoinResult{}, err
	}
	return JoinResult{
		MasterNodeID:   accept.NodeID,
		MasterEndpoint: opts.MasterEndpoint,
		Endpoints:      endpoints,
		TrustedAt:      time.Now().UTC(),
	}, nil
}

func NewEnrollmentHandler(stateDir string, node Node) *EnrollmentHandler {
	return &EnrollmentHandler{
		StateDir: stateDir,
		Node:     node,
		pending:  make(map[string]pendingEnrollment),
	}
}

func (h *EnrollmentHandler) HandleEnrollmentPacket(packet protocol.Packet, remoteEndpoint string) (protocol.Packet, error) {
	switch packet.Type {
	case protocol.PacketTypeEnrollRequest:
		return h.handleEnrollmentRequest(packet.Payload, remoteEndpoint)
	case protocol.PacketTypeEnrollProof:
		return h.handleEnrollmentProof(packet.Payload, remoteEndpoint)
	default:
		return protocol.Packet{}, fmt.Errorf("unsupported enrollment packet type %d", packet.Type)
	}
}

func (h *EnrollmentHandler) handleEnrollmentRequest(payload []byte, remoteEndpoint string) (protocol.Packet, error) {
	var request enrollmentRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return h.reject("", "", "invalid enrollment request"), nil
	}
	now := time.Now().UTC()
	if err := validateEnrollmentRequest(request, now); err != nil {
		return h.reject(request.CodeID, "", err.Error()), nil
	}
	invite, err := loadInvite(h.StateDir, request.CodeID)
	if err != nil {
		return h.reject(request.CodeID, "", "invite not found"), nil
	}
	if err := validateInviteForRequest(invite, request, h.Node, now); err != nil {
		return h.reject(request.CodeID, "", err.Error()), nil
	}
	challengeID, err := randomToken(16)
	if err != nil {
		return protocol.Packet{}, err
	}
	nonce, err := randomToken(24)
	if err != nil {
		return protocol.Packet{}, err
	}
	challenge := enrollmentChallenge{
		Version:             enrollmentVersion,
		CodeID:              request.CodeID,
		ChallengeID:         challengeID,
		ClusterID:           h.Node.ClusterID,
		NodeID:              h.Node.NodeID,
		Role:                h.Node.Role,
		IdentityFingerprint: h.Node.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:      h.Node.PublicIdentity,
		Endpoints:           invite.Endpoints.Normalized(),
		ClientNonce:         request.ClientNonce,
		ChallengeNonce:      nonce,
		SentAt:              now,
	}
	if err := signEnrollmentChallenge(&challenge, h.Node); err != nil {
		return protocol.Packet{}, err
	}
	h.mu.Lock()
	h.pending[challengeID] = pendingEnrollment{
		Request:        request,
		Challenge:      challenge,
		RemoteEndpoint: remoteEndpoint,
		ExpiresAt:      now.Add(enrollmentChallengeTTL),
	}
	h.mu.Unlock()
	return protocol.Packet{Type: protocol.PacketTypeEnrollChallenge, Payload: mustEnrollmentJSON(challenge)}, nil
}

func (h *EnrollmentHandler) handleEnrollmentProof(payload []byte, remoteEndpoint string) (protocol.Packet, error) {
	var proof enrollmentProof
	if err := json.Unmarshal(payload, &proof); err != nil {
		return h.reject("", "", "invalid enrollment proof"), nil
	}
	now := time.Now().UTC()
	if err := validateEnrollmentProof(proof, now); err != nil {
		return h.reject(proof.CodeID, proof.ChallengeID, err.Error()), nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	pending, ok := h.pending[proof.ChallengeID]
	if ok {
		delete(h.pending, proof.ChallengeID)
	}
	if !ok || now.After(pending.ExpiresAt) {
		return h.reject(proof.CodeID, proof.ChallengeID, "enrollment challenge expired"), nil
	}
	if pending.Request.CodeID != proof.CodeID || pending.Request.NodeID != proof.NodeID || pending.Request.Role != proof.Role || pending.Request.IdentityFingerprint != proof.IdentityFingerprint {
		return h.reject(proof.CodeID, proof.ChallengeID, "enrollment proof does not match challenge"), nil
	}
	invite, err := loadInvite(h.StateDir, proof.CodeID)
	if err != nil {
		return h.reject(proof.CodeID, proof.ChallengeID, "invite not found"), nil
	}
	if err := validateInviteForRequest(invite, pending.Request, h.Node, now); err != nil {
		return h.reject(proof.CodeID, proof.ChallengeID, err.Error()), nil
	}
	secretHash, err := base64.RawURLEncoding.DecodeString(invite.SecretHash)
	if err != nil {
		return h.reject(proof.CodeID, proof.ChallengeID, "invite is invalid"), nil
	}
	expected := enrollmentProofValue(secretHash, pending.Request, pending.Challenge)
	if !hmac.Equal([]byte(expected), []byte(proof.Proof)) {
		return h.reject(proof.CodeID, proof.ChallengeID, "join code proof rejected"), nil
	}
	worker := Node{
		Version:        1,
		NodeID:         pending.Request.NodeID,
		Role:           pending.Request.Role,
		ClusterID:      h.Node.ClusterID,
		PublicIdentity: pending.Request.PublicIdentity,
	}
	endpoints := pending.Request.Endpoints.Normalized()
	endpoints.Last = strings.TrimSpace(remoteEndpoint)
	if err := AddTrustWithEndpoints(h.StateDir, worker, endpoints, now); err != nil {
		return protocol.Packet{}, err
	}
	invite.UsedAt = now
	invite.UsedByNodeID = worker.NodeID
	invite.UsedByEndpoint = remoteEndpoint
	if _, err := private.WriteJSONAtomic(inviteFile(h.StateDir, invite.ID), invite); err != nil {
		return protocol.Packet{}, err
	}
	accept := enrollmentAccept{
		Version:                 enrollmentVersion,
		CodeID:                  proof.CodeID,
		ChallengeID:             proof.ChallengeID,
		ClusterID:               h.Node.ClusterID,
		NodeID:                  h.Node.NodeID,
		Role:                    h.Node.Role,
		IdentityFingerprint:     h.Node.PublicIdentity.PublicKeyFingerprint,
		PublicIdentity:          h.Node.PublicIdentity,
		Endpoints:               invite.Endpoints.Normalized(),
		PeerNodeID:              worker.NodeID,
		PeerIdentityFingerprint: worker.PublicIdentity.PublicKeyFingerprint,
		ClientNonce:             pending.Request.ClientNonce,
		ChallengeNonce:          pending.Challenge.ChallengeNonce,
		SentAt:                  now,
	}
	if err := signEnrollmentAccept(&accept, h.Node); err != nil {
		return protocol.Packet{}, err
	}
	return protocol.Packet{Type: protocol.PacketTypeEnrollAccept, Payload: mustEnrollmentJSON(accept)}, nil
}

func (h *EnrollmentHandler) reject(codeID, challengeID, reason string) protocol.Packet {
	return protocol.Packet{
		Type: protocol.PacketTypeEnrollReject,
		Payload: mustEnrollmentJSON(enrollmentReject{
			Version:     enrollmentVersion,
			CodeID:      strings.TrimSpace(codeID),
			ChallengeID: strings.TrimSpace(challengeID),
			Reason:      strings.TrimSpace(reason),
			SentAt:      time.Now().UTC(),
		}),
	}
}

func loadInvite(stateDir, id string) (Invite, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return Invite{}, errors.New("invalid invite id")
	}
	var invite Invite
	if err := private.ReadJSON(inviteFile(stateDir, id), &invite); err != nil {
		return Invite{}, err
	}
	if invite.Version != 1 {
		return Invite{}, fmt.Errorf("unsupported invite version %d", invite.Version)
	}
	return invite, nil
}

func parseInviteCode(code string) (parsedInviteCode, error) {
	parts := strings.Split(strings.TrimSpace(code), ".")
	if len(parts) != 3 || parts[0] != inviteCodePrefix {
		return parsedInviteCode{}, errors.New("join code must use scj1.<payload>.<secret> format")
	}
	if parts[1] == "" || parts[2] == "" {
		return parsedInviteCode{}, errors.New("join code is incomplete")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return parsedInviteCode{}, errors.New("join code payload is invalid")
	}
	if _, err := base64.RawURLEncoding.DecodeString(parts[2]); err != nil {
		return parsedInviteCode{}, errors.New("join code secret is invalid")
	}
	var payload inviteCodePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return parsedInviteCode{}, errors.New("join code payload is invalid")
	}
	if payload.Version != enrollmentVersion {
		return parsedInviteCode{}, fmt.Errorf("unsupported join code version %d", payload.Version)
	}
	if payload.ID == "" || payload.Role == "" || payload.ClusterID == "" || payload.IssuerNodeID == "" || payload.IssuerIdentityFingerprint == "" || payload.ExpiresAt.IsZero() {
		return parsedInviteCode{}, errors.New("join code payload is incomplete")
	}
	return parsedInviteCode{
		Payload:    payload,
		SecretHash: hashInviteSecret(parts[2]),
	}, nil
}

func validateInviteForRequest(invite Invite, request enrollmentRequest, local Node, now time.Time) error {
	if invite.UsedAt.IsZero() == false {
		return errors.New("invite has already been used")
	}
	if invite.ExpiresAt.IsZero() || now.After(invite.ExpiresAt) {
		return errors.New("invite has expired")
	}
	if invite.Role != request.Role {
		return fmt.Errorf("invite accepts %s nodes, got %s", invite.Role, request.Role)
	}
	if invite.ClusterID != local.ClusterID || request.ClusterID != local.ClusterID {
		return errors.New("invite cluster does not match this master")
	}
	if invite.IssuerNodeID != local.NodeID || invite.IssuerIdentityFingerprint != local.PublicIdentity.PublicKeyFingerprint {
		return errors.New("invite issuer does not match this master")
	}
	if local.Role != RoleMaster {
		return errors.New("enrollment requires a master node")
	}
	return nil
}

func validateParsedInviteForJoin(payload inviteCodePayload, node Node, now time.Time) error {
	if payload.ExpiresAt.IsZero() || now.After(payload.ExpiresAt) {
		return errors.New("join code has expired")
	}
	if payload.Role != node.Role {
		return fmt.Errorf("join code accepts %s nodes, local node is %s", payload.Role, node.Role)
	}
	if payload.ClusterID != node.ClusterID {
		return errors.New("join code cluster does not match local node")
	}
	return nil
}

func validateChallengeAgainstInvite(challenge enrollmentChallenge, payload inviteCodePayload, request enrollmentRequest) error {
	if challenge.CodeID != payload.ID {
		return errors.New("enrollment challenge code id mismatch")
	}
	if challenge.ClusterID != payload.ClusterID {
		return errors.New("enrollment challenge cluster mismatch")
	}
	if challenge.NodeID != payload.IssuerNodeID {
		return errors.New("enrollment challenge issuer mismatch")
	}
	if challenge.Role != RoleMaster {
		return errors.New("enrollment challenge issuer is not a master")
	}
	if challenge.IdentityFingerprint != payload.IssuerIdentityFingerprint {
		return errors.New("enrollment challenge master fingerprint mismatch")
	}
	if challenge.ClientNonce != request.ClientNonce {
		return errors.New("enrollment challenge client nonce mismatch")
	}
	return nil
}

func validateAcceptAgainstInvite(accept enrollmentAccept, payload inviteCodePayload, request enrollmentRequest, challenge enrollmentChallenge) error {
	if accept.CodeID != payload.ID {
		return errors.New("enrollment accept code id mismatch")
	}
	if accept.ClusterID != payload.ClusterID {
		return errors.New("enrollment accept cluster mismatch")
	}
	if accept.NodeID != payload.IssuerNodeID {
		return errors.New("enrollment accept issuer mismatch")
	}
	if accept.Role != RoleMaster {
		return errors.New("enrollment accept issuer is not a master")
	}
	if accept.IdentityFingerprint != payload.IssuerIdentityFingerprint {
		return errors.New("enrollment accept master fingerprint mismatch")
	}
	if accept.PeerNodeID != request.NodeID || accept.PeerIdentityFingerprint != request.IdentityFingerprint {
		return errors.New("enrollment accept peer identity mismatch")
	}
	if accept.ClientNonce != request.ClientNonce {
		return errors.New("enrollment accept client nonce mismatch")
	}
	if accept.ChallengeNonce != challenge.ChallengeNonce {
		return errors.New("enrollment accept challenge nonce mismatch")
	}
	return nil
}

func validateEnrollmentRequest(request enrollmentRequest, now time.Time) error {
	if request.Version != enrollmentVersion {
		return fmt.Errorf("unsupported enrollment request version %d", request.Version)
	}
	if request.CodeID == "" || request.ClusterID == "" || request.NodeID == "" || request.Role == "" || request.IdentityFingerprint == "" || request.ClientNonce == "" {
		return errors.New("enrollment request is missing required fields")
	}
	if err := validateEnrollmentTimestamp(now, request.SentAt); err != nil {
		return err
	}
	_, err := identity.PublicKeyFromIdentity(request.PublicIdentity, request.NodeID, request.IdentityFingerprint)
	return err
}

func validateEnrollmentChallenge(challenge enrollmentChallenge, now time.Time) error {
	if challenge.Version != enrollmentVersion {
		return fmt.Errorf("unsupported enrollment challenge version %d", challenge.Version)
	}
	if challenge.CodeID == "" || challenge.ChallengeID == "" || challenge.ClusterID == "" || challenge.NodeID == "" || challenge.Role == "" || challenge.IdentityFingerprint == "" || challenge.ClientNonce == "" || challenge.ChallengeNonce == "" || challenge.Signature == "" {
		return errors.New("enrollment challenge is missing required fields")
	}
	if err := validateEnrollmentTimestamp(now, challenge.SentAt); err != nil {
		return err
	}
	_, err := identity.PublicKeyFromIdentity(challenge.PublicIdentity, challenge.NodeID, challenge.IdentityFingerprint)
	return err
}

func validateEnrollmentProof(proof enrollmentProof, now time.Time) error {
	if proof.Version != enrollmentVersion {
		return fmt.Errorf("unsupported enrollment proof version %d", proof.Version)
	}
	if proof.CodeID == "" || proof.ChallengeID == "" || proof.NodeID == "" || proof.Role == "" || proof.IdentityFingerprint == "" || proof.Proof == "" {
		return errors.New("enrollment proof is missing required fields")
	}
	return validateEnrollmentTimestamp(now, proof.SentAt)
}

func validateEnrollmentAccept(accept enrollmentAccept, now time.Time) error {
	if accept.Version != enrollmentVersion {
		return fmt.Errorf("unsupported enrollment accept version %d", accept.Version)
	}
	if accept.CodeID == "" || accept.ChallengeID == "" || accept.ClusterID == "" || accept.NodeID == "" || accept.Role == "" || accept.IdentityFingerprint == "" || accept.PeerNodeID == "" || accept.PeerIdentityFingerprint == "" || accept.ClientNonce == "" || accept.ChallengeNonce == "" || accept.Signature == "" {
		return errors.New("enrollment accept is missing required fields")
	}
	if err := validateEnrollmentTimestamp(now, accept.SentAt); err != nil {
		return err
	}
	_, err := identity.PublicKeyFromIdentity(accept.PublicIdentity, accept.NodeID, accept.IdentityFingerprint)
	return err
}

func validateEnrollmentTimestamp(now, value time.Time) error {
	if value.IsZero() {
		return errors.New("enrollment timestamp is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	delta := now.UTC().Sub(value.UTC())
	if delta < 0 {
		delta = -delta
	}
	if delta > enrollmentSkew {
		return fmt.Errorf("enrollment timestamp skew exceeds %s", enrollmentSkew)
	}
	return nil
}

func signEnrollmentChallenge(challenge *enrollmentChallenge, node Node) error {
	local, err := node.LocalNode()
	if err != nil {
		return err
	}
	signature := ed25519.Sign(local.PrivateIdentity, enrollmentChallengePayload(*challenge))
	challenge.Signature = base64.RawURLEncoding.EncodeToString(signature)
	return nil
}

func verifyEnrollmentChallenge(challenge enrollmentChallenge) error {
	publicKey, err := identity.PublicKeyFromIdentity(challenge.PublicIdentity, challenge.NodeID, challenge.IdentityFingerprint)
	if err != nil {
		return err
	}
	signature, err := base64.RawURLEncoding.DecodeString(challenge.Signature)
	if err != nil {
		return errors.New("enrollment challenge signature is invalid")
	}
	if !ed25519.Verify(publicKey, enrollmentChallengePayload(challenge), signature) {
		return errors.New("enrollment challenge signature rejected")
	}
	return nil
}

func signEnrollmentAccept(accept *enrollmentAccept, node Node) error {
	local, err := node.LocalNode()
	if err != nil {
		return err
	}
	signature := ed25519.Sign(local.PrivateIdentity, enrollmentAcceptPayload(*accept))
	accept.Signature = base64.RawURLEncoding.EncodeToString(signature)
	return nil
}

func verifyEnrollmentAccept(accept enrollmentAccept) error {
	publicKey, err := identity.PublicKeyFromIdentity(accept.PublicIdentity, accept.NodeID, accept.IdentityFingerprint)
	if err != nil {
		return err
	}
	signature, err := base64.RawURLEncoding.DecodeString(accept.Signature)
	if err != nil {
		return errors.New("enrollment accept signature is invalid")
	}
	if !ed25519.Verify(publicKey, enrollmentAcceptPayload(accept), signature) {
		return errors.New("enrollment accept signature rejected")
	}
	return nil
}

func enrollmentChallengePayload(challenge enrollmentChallenge) []byte {
	challenge.Signature = ""
	return mustEnrollmentJSON(challenge)
}

func enrollmentAcceptPayload(accept enrollmentAccept) []byte {
	accept.Signature = ""
	return mustEnrollmentJSON(accept)
}

func enrollmentProofValue(secretHash []byte, request enrollmentRequest, challenge enrollmentChallenge) string {
	mac := hmac.New(sha256.New, secretHash)
	mac.Write(mustEnrollmentJSON(enrollmentProofTranscript{
		Version:                   enrollmentVersion,
		CodeID:                    request.CodeID,
		ChallengeID:               challenge.ChallengeID,
		ClusterID:                 challenge.ClusterID,
		WorkerNodeID:              request.NodeID,
		WorkerRole:                request.Role,
		WorkerIdentityFingerprint: request.IdentityFingerprint,
		WorkerPublicKey:           request.PublicIdentity.PublicKey,
		WorkerEndpoints:           request.Endpoints.Normalized(),
		MasterNodeID:              challenge.NodeID,
		MasterRole:                challenge.Role,
		MasterIdentityFingerprint: challenge.IdentityFingerprint,
		MasterPublicKey:           challenge.PublicIdentity.PublicKey,
		MasterEndpoints:           challenge.Endpoints.Normalized(),
		ClientNonce:               request.ClientNonce,
		ChallengeNonce:            challenge.ChallengeNonce,
	}))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hashInviteSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func randomToken(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func writeEnrollmentPacket(conn *net.UDPConn, addr *net.UDPAddr, packet protocol.Packet) error {
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(data, addr)
	return err
}

func readEnrollmentPacket(ctx context.Context, conn *net.UDPConn) (protocol.Packet, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	}
	buffer := make([]byte, enrollmentMaxPacket)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil {
				return protocol.Packet{}, ctx.Err()
			}
			if os.IsTimeout(err) {
				return protocol.Packet{}, errors.New("enrollment packet read timed out")
			}
			return protocol.Packet{}, err
		}
		packet, err := protocol.Decode(buffer[:n])
		if err != nil {
			continue
		}
		return packet, nil
	}
}

func decodeEnrollmentReject(packet protocol.Packet) error {
	var reject enrollmentReject
	if err := json.Unmarshal(packet.Payload, &reject); err != nil {
		return fmt.Errorf("parse enrollment reject: %w", err)
	}
	if strings.TrimSpace(reject.Reason) == "" {
		reject.Reason = "enrollment rejected"
	}
	return errors.New(reject.Reason)
}

func validateEndpointSet(endpoints EndpointSet) error {
	endpoints = endpoints.Normalized()
	for label, endpoint := range map[string]string{
		"public endpoint": endpoints.Public,
		"vpc endpoint":    endpoints.VPC,
		"endpoint":        endpoints.Last,
	} {
		if endpoint == "" {
			continue
		}
		if _, err := net.ResolveUDPAddr("udp", endpoint); err != nil {
			return fmt.Errorf("invalid %s %q: %w", label, endpoint, err)
		}
	}
	return nil
}

func mustEnrollmentJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func inviteDir(stateDir string) string {
	return filepath.Join(stateDir, "lab", "invites")
}

func inviteFile(stateDir, id string) string {
	return filepath.Join(inviteDir(stateDir), id+".json")
}

var _ transport.EnrollmentHandler = (*EnrollmentHandler)(nil)
