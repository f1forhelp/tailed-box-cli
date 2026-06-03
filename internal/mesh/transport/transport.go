package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tailedbox/tailedbox/internal/config"
	"github.com/tailedbox/tailedbox/internal/identity"
	meshcrypto "github.com/tailedbox/tailedbox/internal/mesh/crypto"
	"github.com/tailedbox/tailedbox/internal/mesh/protocol"
	"github.com/tailedbox/tailedbox/internal/mesh/session"
	"github.com/tailedbox/tailedbox/internal/mesh/store"
	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	DefaultPingTimeout = 4 * time.Second
	handshakeSkew      = 5 * time.Minute
	maxUDPPacketSize   = protocol.HeaderLen + protocol.MaxPayloadSize
)

type Options struct {
	ListenHost    string
	ListenUDPPort int
	Logger        *slog.Logger
	Now           func() time.Time
}

type Transport struct {
	cfg           *config.Config
	localIdentity identity.PublicIdentity
	privateKey    ed25519.PrivateKey
	conn          *net.UDPConn
	logger        *slog.Logger
	now           func() time.Time
	pendingMu     sync.Mutex
	pending       map[protocol.SessionID]*serverSession
	activeMu      sync.Mutex
	active        map[protocol.SessionID]*serverSession
	closeOnce     sync.Once
	closed        chan struct{}
	boundEndpoint string
	boundUDPPort  int
}

type serverSession struct {
	sessionID   protocol.SessionID
	remoteAddr  *net.UDPAddr
	hello       clientHello
	response    serverHello
	transcript  meshcrypto.Transcript
	clientKey   ed25519.PublicKey
	sender      *session.Sender
	receiver    *session.Receiver
	established time.Time
}

type trustedNodeRecord struct {
	NodeID                  string    `json:"node_id"`
	Role                    string    `json:"role"`
	IdentityFingerprint     string    `json:"identity_fingerprint"`
	PublicKey               string    `json:"public_key"`
	ClusterID               string    `json:"cluster_id"`
	TrustState              string    `json:"trust_state"`
	ReconnectLeaseExpiresAt time.Time `json:"reconnect_lease_expires_at"`
}

type joinedClusterRecord struct {
	ClusterID                 string `json:"cluster_id"`
	MasterNodeID              string `json:"master_node_id"`
	MasterIdentityFingerprint string `json:"master_identity_fingerprint"`
}

func Start(ctx context.Context, cfg *config.Config, opts Options) (*Transport, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.Node.ID == "" || cfg.Node.Role == "" {
		return nil, errors.New("node must be initialized before starting mesh transport")
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	privateKey, _, err := identity.LoadPrivateKey(cfg.Paths.IdentityPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load local mesh identity private key: %w", err)
	}
	localIdentity, err := identity.LoadPublicIdentity(cfg.Paths.IdentityPublicKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load local mesh public identity: %w", err)
	}
	listenHost := opts.ListenHost
	if listenHost == "" {
		listenHost = "0.0.0.0"
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(listenHost, strconv.Itoa(opts.ListenUDPPort)))
	if err != nil {
		return nil, fmt.Errorf("resolve mesh UDP listen address: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on mesh UDP socket: %w", err)
	}
	transport := &Transport{
		cfg:           cfg,
		localIdentity: localIdentity,
		privateKey:    privateKey,
		conn:          conn,
		logger:        opts.Logger,
		now:           now,
		pending:       make(map[protocol.SessionID]*serverSession),
		active:        make(map[protocol.SessionID]*serverSession),
		closed:        make(chan struct{}),
	}
	transport.boundEndpoint = conn.LocalAddr().String()
	if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		transport.boundUDPPort = udpAddr.Port
	}
	go transport.serve(ctx)
	go func() {
		<-ctx.Done()
		_ = transport.Close()
	}()
	return transport, nil
}

func (t *Transport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.closed)
		err = t.conn.Close()
	})
	return err
}

func (t *Transport) BoundEndpoint() string {
	if t == nil {
		return ""
	}
	return t.boundEndpoint
}

func (t *Transport) BoundUDPPort() int {
	if t == nil {
		return 0
	}
	return t.boundUDPPort
}

func (t *Transport) Ping(ctx context.Context, peerNodeID, endpoint string) (time.Duration, error) {
	if t == nil {
		return 0, errors.New("mesh transport is not running")
	}
	peerNodeID = strings.TrimSpace(peerNodeID)
	endpoint = strings.TrimSpace(endpoint)
	if peerNodeID == "" {
		return 0, errors.New("peer node id is required")
	}
	if endpoint == "" {
		return 0, errors.New("peer endpoint is required")
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultPingTimeout)
	defer cancel()
	remoteAddr, err := net.ResolveUDPAddr("udp", endpoint)
	if err != nil {
		return 0, fmt.Errorf("resolve mesh peer endpoint: %w", err)
	}
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return 0, fmt.Errorf("open mesh ping UDP socket: %w", err)
	}
	defer conn.Close()

	initiatorPrivate, initiatorPublic, err := meshcrypto.GenerateEphemeral()
	if err != nil {
		return 0, err
	}
	nonce, err := randomBytes(32)
	if err != nil {
		return 0, err
	}
	now := t.now()
	hello := clientHello{
		Version:              handshakeVersion,
		Mode:                 modeEnrolled,
		ClusterID:            t.cfg.Cluster.ID,
		NodeID:               t.cfg.Node.ID,
		Role:                 t.cfg.Node.Role,
		IdentityFingerprint:  t.localIdentity.PublicKeyFingerprint,
		PublicIdentity:       t.localIdentity,
		EphemeralPublic:      initiatorPublic,
		Nonce:                nonce,
		SentAt:               now,
		SupportedCipherSuite: []string{meshcrypto.CipherSuiteAES256GCM},
	}
	if err := hello.validate(now); err != nil {
		return 0, err
	}
	if err := t.writePacket(conn, remoteAddr, protocol.Packet{Type: protocol.PacketTypeClientHello, Payload: mustJSON(hello)}); err != nil {
		return 0, err
	}

	serverPacket, err := readPacket(ctx, conn)
	if err != nil {
		return 0, err
	}
	if serverPacket.Type != protocol.PacketTypeServerHello {
		return 0, fmt.Errorf("expected server hello, got packet type %d", serverPacket.Type)
	}
	var response serverHello
	if err := json.Unmarshal(serverPacket.Payload, &response); err != nil {
		return 0, fmt.Errorf("parse server hello: %w", err)
	}
	if err := response.validate(t.now()); err != nil {
		return 0, err
	}
	if response.NodeID != peerNodeID {
		return 0, fmt.Errorf("server hello came from node %s, expected %s", response.NodeID, peerNodeID)
	}
	if err := t.validateResponderTrust(response); err != nil {
		return 0, err
	}
	responderKey, err := publicKeyFromIdentity(response.PublicIdentity, response.NodeID, response.IdentityFingerprint)
	if err != nil {
		return 0, err
	}
	transcript := buildTranscript(hello, response)
	if !meshcrypto.VerifyTranscript(responderKey, transcript, response.Signature) {
		return 0, errors.New("server hello signature rejected")
	}
	keys, err := meshcrypto.DeriveSessionKeys(initiatorPrivate, response.EphemeralPublic, transcript)
	if err != nil {
		return 0, err
	}
	sender, err := session.NewSender(response.SessionID, keys.InitiatorToResponderKey, keys.InitiatorNoncePrefix)
	if err != nil {
		return 0, err
	}
	receiver, err := session.NewReceiver(response.SessionID, keys.ResponderToInitiatorKey, keys.ResponderNoncePrefix, nil)
	if err != nil {
		return 0, err
	}
	signature, err := meshcrypto.SignTranscript(t.privateKey, transcript)
	if err != nil {
		return 0, err
	}
	authPacket, err := sender.Seal(protocol.PacketTypeClientAuth, mustJSON(clientAuth{
		Version:             handshakeVersion,
		ClusterID:           hello.ClusterID,
		NodeID:              hello.NodeID,
		Role:                hello.Role,
		IdentityFingerprint: hello.IdentityFingerprint,
		SentAt:              t.now(),
		Signature:           signature,
	}))
	if err != nil {
		return 0, err
	}
	if err := t.writePacket(conn, remoteAddr, authPacket); err != nil {
		return 0, err
	}
	for {
		packet, err := readPacket(ctx, conn)
		if err != nil {
			return 0, err
		}
		if packet.Type != protocol.PacketTypeEncryptedData || packet.SessionID != response.SessionID {
			continue
		}
		plaintext, err := receiver.Open(packet)
		if err != nil {
			return 0, err
		}
		message, err := protocol.DecodeControlMessage(plaintext)
		if err != nil {
			return 0, err
		}
		if message.Type == protocol.MessageTypeStatusResponse && message.ID == "session_established" {
			break
		}
	}

	messageID, err := newMessageID()
	if err != nil {
		return 0, err
	}
	pingMessage, err := protocol.EncodeControlMessage(protocol.ControlMessage{
		Type:       protocol.MessageTypePing,
		ID:         messageID,
		NodeID:     t.cfg.Node.ID,
		PeerNodeID: peerNodeID,
		SentAt:     t.now(),
	})
	if err != nil {
		return 0, err
	}
	pingPacket, err := sender.Seal(protocol.PacketTypeEncryptedData, pingMessage)
	if err != nil {
		return 0, err
	}
	startedAt := time.Now()
	if err := t.writePacket(conn, remoteAddr, pingPacket); err != nil {
		return 0, err
	}
	for {
		packet, err := readPacket(ctx, conn)
		if err != nil {
			return 0, err
		}
		if packet.Type != protocol.PacketTypeEncryptedData || packet.SessionID != response.SessionID {
			continue
		}
		plaintext, err := receiver.Open(packet)
		if err != nil {
			return 0, err
		}
		message, err := protocol.DecodeControlMessage(plaintext)
		if err != nil {
			return 0, err
		}
		if message.Type != protocol.MessageTypePong || message.ID != messageID {
			continue
		}
		latency := time.Since(startedAt)
		_ = t.writePeer(response.NodeID, response.Role, response.IdentityFingerprint, endpoint, store.SessionStateConnected)
		return latency, nil
	}
}

func (t *Transport) serve(ctx context.Context) {
	buffer := make([]byte, maxUDPPacketSize)
	for {
		n, addr, err := t.conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-t.closed:
				return
			case <-ctx.Done():
				return
			default:
			}
			continue
		}
		data := append([]byte(nil), buffer[:n]...)
		go t.handleDatagram(data, addr)
	}
}

func (t *Transport) handleDatagram(data []byte, addr *net.UDPAddr) {
	packet, err := protocol.Decode(data)
	if err != nil {
		t.debug("mesh UDP packet rejected", "remote", addr.String(), "error", err)
		return
	}
	var handleErr error
	switch packet.Type {
	case protocol.PacketTypeClientHello:
		handleErr = t.handleClientHello(packet, addr)
	case protocol.PacketTypeClientAuth:
		handleErr = t.handleClientAuth(packet, addr)
	case protocol.PacketTypeEncryptedData:
		handleErr = t.handleEncryptedData(packet, addr)
	}
	if handleErr != nil {
		t.debug("mesh UDP packet handling failed", "remote", addr.String(), "type", int(packet.Type), "error", handleErr)
	}
}

func (t *Transport) handleClientHello(packet protocol.Packet, addr *net.UDPAddr) error {
	var hello clientHello
	if err := json.Unmarshal(packet.Payload, &hello); err != nil {
		return fmt.Errorf("parse client hello: %w", err)
	}
	if err := hello.validate(t.now()); err != nil {
		return err
	}
	if err := t.validateInitiatorTrust(hello); err != nil {
		return err
	}
	clientKey, err := publicKeyFromIdentity(hello.PublicIdentity, hello.NodeID, hello.IdentityFingerprint)
	if err != nil {
		return err
	}
	responderPrivate, responderPublic, err := meshcrypto.GenerateEphemeral()
	if err != nil {
		return err
	}
	nonce, err := randomBytes(32)
	if err != nil {
		return err
	}
	sessionID, err := newSessionID()
	if err != nil {
		return err
	}
	response := serverHello{
		Version:             handshakeVersion,
		Mode:                modeEnrolled,
		ClusterID:           hello.ClusterID,
		NodeID:              t.cfg.Node.ID,
		Role:                t.cfg.Node.Role,
		IdentityFingerprint: t.localIdentity.PublicKeyFingerprint,
		PublicIdentity:      t.localIdentity,
		EphemeralPublic:     responderPublic,
		Nonce:               nonce,
		SentAt:              t.now(),
		CipherSuite:         meshcrypto.CipherSuiteAES256GCM,
		SessionID:           sessionID,
	}
	transcript := buildTranscript(hello, response)
	signature, err := meshcrypto.SignTranscript(t.privateKey, transcript)
	if err != nil {
		return err
	}
	response.Signature = signature
	keys, err := meshcrypto.DeriveSessionKeys(responderPrivate, hello.EphemeralPublic, transcript)
	if err != nil {
		return err
	}
	sender, err := session.NewSender(sessionID, keys.ResponderToInitiatorKey, keys.ResponderNoncePrefix)
	if err != nil {
		return err
	}
	receiver, err := session.NewReceiver(sessionID, keys.InitiatorToResponderKey, keys.InitiatorNoncePrefix, nil)
	if err != nil {
		return err
	}
	serverSession := &serverSession{
		sessionID:  sessionID,
		remoteAddr: addr,
		hello:      hello,
		response:   response,
		transcript: transcript,
		clientKey:  clientKey,
		sender:     sender,
		receiver:   receiver,
	}
	t.pendingMu.Lock()
	t.pending[sessionID] = serverSession
	t.pendingMu.Unlock()
	return t.writePacket(t.conn, addr, protocol.Packet{Type: protocol.PacketTypeServerHello, Payload: mustJSON(response)})
}

func (t *Transport) handleClientAuth(packet protocol.Packet, addr *net.UDPAddr) error {
	pending := t.takePending(packet.SessionID)
	if pending == nil {
		return errors.New("client auth has no pending session")
	}
	plaintext, err := pending.receiver.Open(packet)
	if err != nil {
		return err
	}
	var auth clientAuth
	if err := json.Unmarshal(plaintext, &auth); err != nil {
		return fmt.Errorf("parse client auth: %w", err)
	}
	if err := auth.validate(t.now()); err != nil {
		return err
	}
	if auth.ClusterID != pending.hello.ClusterID || auth.NodeID != pending.hello.NodeID || auth.Role != pending.hello.Role || auth.IdentityFingerprint != pending.hello.IdentityFingerprint {
		return errors.New("client auth does not match client hello")
	}
	if !meshcrypto.VerifyTranscript(pending.clientKey, pending.transcript, auth.Signature) {
		return errors.New("client auth signature rejected")
	}
	pending.established = t.now()
	pending.remoteAddr = addr
	t.activeMu.Lock()
	t.active[pending.sessionID] = pending
	t.activeMu.Unlock()
	_ = t.writePeer(pending.hello.NodeID, pending.hello.Role, pending.hello.IdentityFingerprint, addr.String(), store.SessionStateConnected)
	ack, err := protocol.EncodeControlMessage(protocol.ControlMessage{
		Type:       protocol.MessageTypeStatusResponse,
		ID:         "session_established",
		NodeID:     t.cfg.Node.ID,
		PeerNodeID: pending.hello.NodeID,
		SentAt:     t.now(),
	})
	if err != nil {
		return err
	}
	ackPacket, err := pending.sender.Seal(protocol.PacketTypeEncryptedData, ack)
	if err != nil {
		return err
	}
	return t.writePacket(t.conn, addr, ackPacket)
}

func (t *Transport) handleEncryptedData(packet protocol.Packet, addr *net.UDPAddr) error {
	active := t.activeSession(packet.SessionID)
	if active == nil {
		return errors.New("encrypted data has no active session")
	}
	plaintext, err := active.receiver.Open(packet)
	if err != nil {
		return err
	}
	message, err := protocol.DecodeControlMessage(plaintext)
	if err != nil {
		return err
	}
	if message.Type != protocol.MessageTypePing {
		return nil
	}
	pong, err := protocol.EncodeControlMessage(protocol.ControlMessage{
		Type:       protocol.MessageTypePong,
		ID:         message.ID,
		NodeID:     t.cfg.Node.ID,
		PeerNodeID: message.NodeID,
		SentAt:     t.now(),
	})
	if err != nil {
		return err
	}
	responsePacket, err := active.sender.Seal(protocol.PacketTypeEncryptedData, pong)
	if err != nil {
		return err
	}
	active.remoteAddr = addr
	_ = t.writePeer(active.hello.NodeID, active.hello.Role, active.hello.IdentityFingerprint, addr.String(), store.SessionStateConnected)
	return t.writePacket(t.conn, addr, responsePacket)
}

func (t *Transport) validateInitiatorTrust(hello clientHello) error {
	if hello.ClusterID != t.cfg.Cluster.ID {
		return fmt.Errorf("client hello cluster %s does not match local cluster %s", hello.ClusterID, t.cfg.Cluster.ID)
	}
	switch t.cfg.Node.Role {
	case config.RoleMaster:
		trusted, err := readTrustedNode(t.cfg, hello.NodeID)
		if err != nil {
			return fmt.Errorf("mesh peer is not trusted: %w", err)
		}
		if trusted.TrustState != "trusted" {
			return fmt.Errorf("mesh peer %s is not trusted", hello.NodeID)
		}
		if trusted.Role != hello.Role || trusted.ClusterID != hello.ClusterID || trusted.IdentityFingerprint != hello.IdentityFingerprint || trusted.PublicKey != hello.PublicIdentity.PublicKey {
			return errors.New("mesh peer identity does not match trusted-node record")
		}
		if !trusted.ReconnectLeaseExpiresAt.IsZero() && !t.now().Before(trusted.ReconnectLeaseExpiresAt) {
			return fmt.Errorf("mesh peer reconnect lease has expired")
		}
		return nil
	case config.RoleWorker:
		joined, err := readJoinedCluster(t.cfg)
		if err != nil {
			return err
		}
		if hello.NodeID != joined.MasterNodeID || hello.IdentityFingerprint != joined.MasterIdentityFingerprint || hello.ClusterID != joined.ClusterID || hello.Role != config.RoleMaster {
			return errors.New("mesh initiator does not match pinned master identity")
		}
		return nil
	default:
		return fmt.Errorf("unsupported local mesh role %q", t.cfg.Node.Role)
	}
}

func (t *Transport) validateResponderTrust(response serverHello) error {
	if response.ClusterID != t.cfg.Cluster.ID {
		return fmt.Errorf("server hello cluster %s does not match local cluster %s", response.ClusterID, t.cfg.Cluster.ID)
	}
	switch t.cfg.Node.Role {
	case config.RoleWorker:
		joined, err := readJoinedCluster(t.cfg)
		if err != nil {
			return err
		}
		if response.NodeID != joined.MasterNodeID || response.IdentityFingerprint != joined.MasterIdentityFingerprint || response.Role != config.RoleMaster {
			return errors.New("server hello does not match pinned master identity")
		}
		return nil
	case config.RoleMaster:
		trusted, err := readTrustedNode(t.cfg, response.NodeID)
		if err != nil {
			return err
		}
		if trusted.Role != response.Role || trusted.IdentityFingerprint != response.IdentityFingerprint || trusted.PublicKey != response.PublicIdentity.PublicKey {
			return errors.New("server hello does not match trusted-node record")
		}
		return nil
	default:
		return fmt.Errorf("unsupported local mesh role %q", t.cfg.Node.Role)
	}
}

func (t *Transport) writePacket(conn *net.UDPConn, addr *net.UDPAddr, packet protocol.Packet) error {
	data, err := protocol.Encode(packet)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(data, addr)
	return err
}

func (t *Transport) takePending(sessionID protocol.SessionID) *serverSession {
	t.pendingMu.Lock()
	defer t.pendingMu.Unlock()
	pending := t.pending[sessionID]
	delete(t.pending, sessionID)
	return pending
}

func (t *Transport) activeSession(sessionID protocol.SessionID) *serverSession {
	t.activeMu.Lock()
	defer t.activeMu.Unlock()
	return t.active[sessionID]
}

func (t *Transport) writePeer(nodeID, role, fingerprint, endpoint, sessionState string) error {
	_, err := store.WritePeer(t.cfg.Paths, store.PeerObservation{
		NodeID:              nodeID,
		Role:                role,
		IdentityFingerprint: fingerprint,
		LastEndpoint:        endpoint,
		LastSeenAt:          t.now(),
		SessionState:        sessionState,
	})
	return err
}

func readPacket(ctx context.Context, conn *net.UDPConn) (protocol.Packet, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	}
	buffer := make([]byte, maxUDPPacketSize)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil {
				return protocol.Packet{}, ctx.Err()
			}
			if os.IsTimeout(err) {
				return protocol.Packet{}, fmt.Errorf("mesh packet read timed out")
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

func validateTimestamp(now, value time.Time) error {
	if value.IsZero() {
		return errors.New("mesh handshake timestamp is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	delta := now.UTC().Sub(value.UTC())
	if delta < 0 {
		delta = -delta
	}
	if delta > handshakeSkew {
		return fmt.Errorf("mesh handshake timestamp skew exceeds %s", handshakeSkew)
	}
	return nil
}

func newSessionID() (protocol.SessionID, error) {
	var sessionID protocol.SessionID
	if _, err := rand.Read(sessionID[:]); err != nil {
		return protocol.SessionID{}, fmt.Errorf("generate mesh session id: %w", err)
	}
	return sessionID, nil
}

func randomBytes(size int) ([]byte, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return nil, fmt.Errorf("generate mesh random bytes: %w", err)
	}
	return data, nil
}

func newMessageID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate mesh message id: %w", err)
	}
	return "msg_" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])), nil
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func (t *Transport) debug(message string, args ...any) {
	if t != nil && t.logger != nil {
		t.logger.Debug(message, args...)
	}
}

func readTrustedNode(cfg *config.Config, nodeID string) (trustedNodeRecord, error) {
	if nodeID == "" || nodeID == "." || nodeID == ".." || strings.ContainsAny(nodeID, `/\`) {
		return trustedNodeRecord{}, fmt.Errorf("invalid trusted node id %q", nodeID)
	}
	var record trustedNodeRecord
	if err := secrets.ReadJSON(filepath.Join(cfg.Paths.TrustedNodesDir, nodeID+".json"), &record); err != nil {
		return trustedNodeRecord{}, err
	}
	return record, nil
}

func readJoinedCluster(cfg *config.Config) (joinedClusterRecord, error) {
	var record joinedClusterRecord
	if err := secrets.ReadJSON(cfg.Paths.JoinedClusterFile, &record); err != nil {
		return joinedClusterRecord{}, err
	}
	return record, nil
}
