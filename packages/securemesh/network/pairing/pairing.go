package pairing

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/join"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/network/tlsidentity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

const (
	ALPN                    = "tailed-box-pairing/1"
	DefaultBind             = "127.0.0.1:9444"
	DefaultPort             = "9444"
	MaxFrameSize            = 64 * 1024
	messageTypePairRequest  = "pair_request"
	messageTypePairResponse = "pair_response"
	messageVersion          = 1
)

var (
	ErrLocalStateMismatch = errors.New("local identity and network state mismatch")
	ErrUnauthorized       = errors.New("pairing listener requires a master identity")
	ErrFrameTooLarge      = errors.New("frame too large")
	ErrInvalidMessage     = errors.New("invalid pairing message")
	ErrMasterMismatch     = errors.New("master identity did not match expected node")
)

type PublicPeer struct {
	NodeID     identity.NodeID       `json:"node_id"`
	NetworkID  identity.NetworkID    `json:"network_id"`
	Role       identity.Role         `json:"role"`
	PublicKeys identity.PublicKeySet `json:"public_keys"`
}

func PublicPeerFromIdentity(local identity.Identity) PublicPeer {
	return PublicPeer{
		NodeID:     local.NodeID,
		NetworkID:  local.NetworkID,
		Role:       local.Role,
		PublicKeys: local.PublicKeys,
	}
}

func (p PublicPeer) Validate() error {
	if err := p.NodeID.Validate(); err != nil {
		return err
	}
	if err := p.NetworkID.Validate(); err != nil {
		return err
	}
	if err := p.Role.Validate(); err != nil {
		return err
	}
	if err := p.PublicKeys.Validate(); err != nil {
		return err
	}
	derived, err := identity.DeriveNodeID(p.PublicKeys)
	if err != nil {
		return err
	}
	if derived != p.NodeID {
		return fmt.Errorf("%w: peer node id does not match public keys", ErrInvalidMessage)
	}
	return nil
}

type Server struct {
	Paths config.Paths
}

func NewServer(paths config.Paths) Server {
	return Server{Paths: paths}
}

func ListenAndServe(ctx context.Context, paths config.Paths, bind string) error {
	listener, err := NewServer(paths).Listen(bind)
	if err != nil {
		return err
	}
	defer listener.Close()
	return listener.Serve(ctx)
}

func (s Server) Listen(bind string) (*Listener, error) {
	if strings.TrimSpace(bind) == "" {
		bind = DefaultBind
	}
	tlsConfig, local, err := serverTLSConfig(s.Paths)
	if err != nil {
		return nil, err
	}
	listener, err := tls.Listen("tcp", bind, tlsConfig)
	if err != nil {
		return nil, err
	}
	return &Listener{listener: listener, paths: s.Paths, local: local}, nil
}

type Listener struct {
	listener net.Listener
	paths    config.Paths
	local    identity.Identity
}

func (l *Listener) Addr() net.Addr {
	if l == nil || l.listener == nil {
		return nil
	}
	return l.listener.Addr()
}

func (l *Listener) Close() error {
	if l == nil || l.listener == nil {
		return nil
	}
	return l.listener.Close()
}

func (l *Listener) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	for {
		conn, err := l.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go func() {
			_ = l.handle(conn)
		}()
	}
}

func (l *Listener) handle(conn net.Conn) error {
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return errors.New("accepted connection is not tls")
	}
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	if tlsConn.ConnectionState().NegotiatedProtocol != ALPN {
		return fmt.Errorf("%w: negotiated ALPN %q", ErrInvalidMessage, tlsConn.ConnectionState().NegotiatedProtocol)
	}
	request, err := readMessage[pairRequest](tlsConn)
	if err != nil {
		return err
	}
	response, err := l.accept(request)
	if err != nil {
		return err
	}
	return writeMessage(tlsConn, response)
}

func (l *Listener) accept(request pairRequest) (pairResponse, error) {
	if err := request.Validate(); err != nil {
		return pairResponse{}, err
	}
	if request.Peer.NetworkID != l.local.NetworkID {
		return pairResponse{}, join.ErrWrongNetwork
	}
	if request.Role != request.Peer.Role {
		return pairResponse{}, join.ErrWrongRole
	}
	revoked, err := revocation.NewStore(l.paths).IsRevoked(request.Peer.NodeID)
	if err != nil {
		return pairResponse{}, err
	}
	if revoked {
		return pairResponse{}, tlsidentity.ErrRevokedPeer
	}

	peerStore := peer.NewStore(l.paths)
	if existing, err := peerStore.Get(request.Peer.NodeID); err == nil {
		if !samePeer(existing, request.Peer) {
			return pairResponse{}, peer.ErrPeerExists
		}
	} else if !errors.Is(err, peer.ErrPeerNotFound) {
		return pairResponse{}, err
	}

	if _, err := join.NewStore(l.paths).ValidateAndConsumeWith(join.ConsumeRequest{
		Code:         request.Code,
		NetworkID:    l.local.NetworkID,
		ExpectedRole: request.Role,
		ConsumedBy:   request.Peer.NodeID,
	}, func(join.Record) error {
		return addPeerIfMissing(peerStore, request.Peer, time.Now().UTC())
	}); err != nil {
		return pairResponse{}, err
	}

	return pairResponse{
		Version:      messageVersion,
		Type:         messageTypePairResponse,
		Accepted:     true,
		NetworkID:    l.local.NetworkID,
		Master:       PublicPeerFromIdentity(l.local),
		JoinedNodeID: request.Peer.NodeID,
	}, nil
}

type JoinResult struct {
	LocalNodeID  identity.NodeID
	MasterNodeID identity.NodeID
	NetworkID    identity.NetworkID
	Role         identity.Role
	Endpoint     string
}

func Join(ctx context.Context, paths config.Paths, endpoint, code string, role identity.Role, expectedMaster identity.NodeID) (JoinResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := role.Validate(); err != nil {
		return JoinResult{}, err
	}
	if err := expectedMaster.Validate(); err != nil {
		return JoinResult{}, err
	}
	endpoint, err := NormalizeEndpoint(endpoint)
	if err != nil {
		return JoinResult{}, err
	}

	var master tlsidentity.CertificateMetadata
	tlsConfig := clientTLSConfig(expectedMaster, &master)
	dialer := tls.Dialer{Config: tlsConfig}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return JoinResult{}, err
	}
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return JoinResult{}, errors.New("dialed connection is not tls")
	}
	if tlsConn.ConnectionState().NegotiatedProtocol != ALPN {
		return JoinResult{}, fmt.Errorf("%w: negotiated ALPN %q", ErrInvalidMessage, tlsConn.ConnectionState().NegotiatedProtocol)
	}
	if master.NodeID == "" {
		metadata, err := tlsidentity.MetadataFromCertificate(tlsConn.ConnectionState().PeerCertificates[0])
		if err != nil {
			return JoinResult{}, err
		}
		master = metadata
	}

	local, generated, err := localJoinIdentity(paths, master.NetworkID, role, time.Now().UTC())
	if err != nil {
		return JoinResult{}, err
	}
	request := pairRequest{
		Version: messageVersion,
		Type:    messageTypePairRequest,
		Code:    code,
		Role:    role,
		Peer:    PublicPeerFromIdentity(local),
		Time:    time.Now().UTC(),
	}
	if err := writeMessage(tlsConn, request); err != nil {
		return JoinResult{}, err
	}
	response, err := readMessage[pairResponse](tlsConn)
	if err != nil {
		return JoinResult{}, err
	}
	if err := response.Validate(expectedMaster, local.NodeID, master.NetworkID); err != nil {
		return JoinResult{}, err
	}

	if err := saveJoinedState(paths, local, generated, response.Master, time.Now().UTC()); err != nil {
		return JoinResult{}, err
	}
	return JoinResult{
		LocalNodeID:  local.NodeID,
		MasterNodeID: response.Master.NodeID,
		NetworkID:    response.NetworkID,
		Role:         local.Role,
		Endpoint:     endpoint,
	}, nil
}

func NormalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.New("endpoint is required")
	}
	if _, _, err := net.SplitHostPort(endpoint); err == nil {
		return endpoint, nil
	}
	if strings.Contains(endpoint, ":") && strings.Count(endpoint, ":") > 1 {
		return net.JoinHostPort(endpoint, DefaultPort), nil
	}
	return net.JoinHostPort(endpoint, DefaultPort), nil
}

func serverTLSConfig(paths config.Paths) (*tls.Config, identity.Identity, error) {
	local, err := identity.LoadIdentity(paths)
	if err != nil {
		return nil, identity.Identity{}, err
	}
	if local.Role != identity.RoleMaster {
		return nil, identity.Identity{}, ErrUnauthorized
	}
	network, err := identity.LoadNetwork(paths)
	if err != nil {
		return nil, identity.Identity{}, err
	}
	if local.NetworkID != network.ID {
		return nil, identity.Identity{}, ErrLocalStateMismatch
	}
	certificate, err := tlsidentity.NewCertificate(local, time.Now().UTC())
	if err != nil {
		return nil, identity.Identity{}, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
		NextProtos:   []string{ALPN},
	}, local, nil
}

func clientTLSConfig(expectedMaster identity.NodeID, out *tlsidentity.CertificateMetadata) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{ALPN},
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return tlsidentity.ErrMissingCertificate
			}
			leaf, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("%w: parse certificate: %w", tlsidentity.ErrInvalidCertificate, err)
			}
			metadata, err := tlsidentity.MetadataFromCertificate(leaf)
			if err != nil {
				return err
			}
			if metadata.NodeID != expectedMaster || metadata.Role != identity.RoleMaster {
				return ErrMasterMismatch
			}
			now := time.Now().UTC()
			if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
				return fmt.Errorf("%w: certificate is outside validity window", tlsidentity.ErrInvalidCertificate)
			}
			if out != nil {
				*out = metadata
			}
			return nil
		},
	}
}

func localJoinIdentity(paths config.Paths, networkID identity.NetworkID, role identity.Role, now time.Time) (identity.Identity, bool, error) {
	local, err := identity.LoadIdentity(paths)
	if err == nil {
		if local.NetworkID != networkID {
			return identity.Identity{}, false, ErrLocalStateMismatch
		}
		if local.Role != role {
			return identity.Identity{}, false, join.ErrWrongRole
		}
		return local, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return identity.Identity{}, false, err
	}
	local, err = identity.GenerateIdentity(networkID, role, now)
	return local, true, err
}

func saveJoinedState(paths config.Paths, local identity.Identity, generated bool, master PublicPeer, now time.Time) error {
	if err := ensureNetwork(paths, local.NetworkID, master.NodeID, now); err != nil {
		return err
	}
	if generated {
		if err := identity.SaveIdentity(paths, local); err != nil {
			return err
		}
	}
	return addPeerIfMissing(peer.NewStore(paths), master, now)
}

func ensureNetwork(paths config.Paths, networkID identity.NetworkID, createdBy identity.NodeID, now time.Time) error {
	network, err := identity.LoadNetwork(paths)
	if err == nil {
		if network.ID != networkID {
			return ErrLocalStateMismatch
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return identity.SaveNetwork(paths, identity.Network{
		Version:   identity.NetworkVersion,
		ID:        networkID,
		CreatedAt: now.UTC(),
		CreatedBy: createdBy,
	})
}

func addPeerIfMissing(store peer.Store, public PublicPeer, now time.Time) error {
	if err := public.Validate(); err != nil {
		return err
	}
	existing, err := store.Get(public.NodeID)
	if err == nil {
		if samePeer(existing, public) {
			return nil
		}
		return peer.ErrPeerExists
	}
	if !errors.Is(err, peer.ErrPeerNotFound) {
		return err
	}
	return store.Add(peer.Record{
		NodeID:     public.NodeID,
		NetworkID:  public.NetworkID,
		Role:       public.Role,
		PublicKeys: public.PublicKeys,
		Status:     peer.StatusActive,
		AddedAt:    now.UTC(),
	})
}

func samePeer(record peer.Record, public PublicPeer) bool {
	return record.NodeID == public.NodeID &&
		record.NetworkID == public.NetworkID &&
		record.Role == public.Role &&
		samePublicKeys(record.PublicKeys, public.PublicKeys)
}

func samePublicKeys(a, b identity.PublicKeySet) bool {
	return a.Signing.Algorithm == b.Signing.Algorithm &&
		a.Transport.Algorithm == b.Transport.Algorithm &&
		bytes.Equal(a.Signing.Bytes, b.Signing.Bytes) &&
		bytes.Equal(a.Transport.Bytes, b.Transport.Bytes)
}

type pairRequest struct {
	Version int           `json:"version"`
	Type    string        `json:"type"`
	Code    string        `json:"code"`
	Role    identity.Role `json:"role"`
	Peer    PublicPeer    `json:"peer"`
	Time    time.Time     `json:"time"`
}

func (r pairRequest) Validate() error {
	if r.Version != messageVersion || r.Type != messageTypePairRequest {
		return ErrInvalidMessage
	}
	if strings.TrimSpace(r.Code) == "" {
		return join.ErrInvalidCode
	}
	if err := r.Role.Validate(); err != nil {
		return err
	}
	if err := r.Peer.Validate(); err != nil {
		return err
	}
	if r.Time.IsZero() {
		return fmt.Errorf("%w: time is required", ErrInvalidMessage)
	}
	return nil
}

type pairResponse struct {
	Version      int                `json:"version"`
	Type         string             `json:"type"`
	Accepted     bool               `json:"accepted"`
	NetworkID    identity.NetworkID `json:"network_id"`
	Master       PublicPeer         `json:"master"`
	JoinedNodeID identity.NodeID    `json:"joined_node_id"`
}

func (r pairResponse) Validate(expectedMaster, localNode identity.NodeID, networkID identity.NetworkID) error {
	if r.Version != messageVersion || r.Type != messageTypePairResponse || !r.Accepted {
		return ErrInvalidMessage
	}
	if r.NetworkID != networkID || r.Master.NetworkID != networkID {
		return join.ErrWrongNetwork
	}
	if r.Master.NodeID != expectedMaster || r.Master.Role != identity.RoleMaster {
		return ErrMasterMismatch
	}
	if r.JoinedNodeID != localNode {
		return fmt.Errorf("%w: joined node mismatch", ErrInvalidMessage)
	}
	return r.Master.Validate()
}

func readMessage[T any](reader io.Reader) (T, error) {
	var zero T
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return zero, err
	}
	length := binary.BigEndian.Uint32(header[:])
	if length == 0 || length > MaxFrameSize {
		return zero, fmt.Errorf("%w: %d", ErrFrameTooLarge, length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return zero, err
	}
	var message T
	if err := json.Unmarshal(payload, &message); err != nil {
		return zero, fmt.Errorf("%w: decode: %w", ErrInvalidMessage, err)
	}
	return message, nil
}

func writeMessage(writer io.Writer, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) > MaxFrameSize {
		return fmt.Errorf("%w: %d", ErrFrameTooLarge, len(payload))
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := writer.Write(header[:]); err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}
