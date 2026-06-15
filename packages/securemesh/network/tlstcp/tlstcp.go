package tlstcp

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/network/tlsidentity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/peer"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

const (
	DefaultBind     = "127.0.0.1:9443"
	DefaultPort     = "9443"
	MaxFrameSize    = 64 * 1024
	messageTypePing = "ping"
	messageTypePong = "pong"
)

var (
	ErrLocalStateMismatch = errors.New("local identity and network state mismatch")
	ErrFrameTooLarge      = errors.New("frame too large")
	ErrInvalidMessage     = errors.New("invalid mesh control message")
)

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
	tlsConfig, local, err := buildTLSConfig(s.Paths, "")
	if err != nil {
		return nil, err
	}
	tlsConfig.ClientAuth = tls.RequireAnyClientCert

	listener, err := tls.Listen("tcp", bind, tlsConfig)
	if err != nil {
		return nil, err
	}
	return &Listener{listener: listener, local: local}, nil
}

type Listener struct {
	listener net.Listener
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
	if tlsConn.ConnectionState().NegotiatedProtocol != tlsidentity.ALPN {
		return fmt.Errorf("%w: negotiated ALPN %q", ErrInvalidMessage, tlsConn.ConnectionState().NegotiatedProtocol)
	}
	request, err := readMessage(tlsConn)
	if err != nil {
		return err
	}
	if request.Type != messageTypePing {
		return fmt.Errorf("%w: expected ping, got %q", ErrInvalidMessage, request.Type)
	}
	return writeMessage(tlsConn, wireMessage{
		Type:      messageTypePong,
		NodeID:    l.local.NodeID,
		NetworkID: l.local.NetworkID,
		Role:      l.local.Role,
		Time:      time.Now().UTC(),
	})
}

type PingResult struct {
	LocalNodeID  identity.NodeID
	RemoteNodeID identity.NodeID
	RemoteRole   identity.Role
	NetworkID    identity.NetworkID
	Endpoint     string
	RTT          time.Duration
}

func Ping(ctx context.Context, paths config.Paths, endpoint string) (PingResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint, err := NormalizeEndpoint(endpoint)
	if err != nil {
		return PingResult{}, err
	}
	tlsConfig, local, err := buildTLSConfig(paths, "")
	if err != nil {
		return PingResult{}, err
	}
	dialer := tls.Dialer{Config: tlsConfig}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return PingResult{}, err
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return PingResult{}, errors.New("dialed connection is not tls")
	}
	state := tlsConn.ConnectionState()
	if state.NegotiatedProtocol != tlsidentity.ALPN {
		return PingResult{}, fmt.Errorf("%w: negotiated ALPN %q", ErrInvalidMessage, state.NegotiatedProtocol)
	}
	remote, err := tlsidentity.MetadataFromCertificate(state.PeerCertificates[0])
	if err != nil {
		return PingResult{}, err
	}

	if err := writeMessage(tlsConn, wireMessage{Type: messageTypePing, NodeID: local.NodeID, NetworkID: local.NetworkID, Role: local.Role, Time: time.Now().UTC()}); err != nil {
		return PingResult{}, err
	}
	response, err := readMessage(tlsConn)
	if err != nil {
		return PingResult{}, err
	}
	if response.Type != messageTypePong {
		return PingResult{}, fmt.Errorf("%w: expected pong, got %q", ErrInvalidMessage, response.Type)
	}
	return PingResult{
		LocalNodeID:  local.NodeID,
		RemoteNodeID: remote.NodeID,
		RemoteRole:   remote.Role,
		NetworkID:    remote.NetworkID,
		Endpoint:     endpoint,
		RTT:          time.Since(start),
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

func buildTLSConfig(paths config.Paths, expectedRole identity.Role) (*tls.Config, identity.Identity, error) {
	local, err := identity.LoadIdentity(paths)
	if err != nil {
		return nil, identity.Identity{}, err
	}
	network, err := identity.LoadNetwork(paths)
	if err != nil {
		return nil, identity.Identity{}, err
	}
	if local.NetworkID != network.ID {
		return nil, identity.Identity{}, ErrLocalStateMismatch
	}
	verifier := tlsidentity.Verifier{Options: tlsidentity.VerifyOptions{
		NetworkID:    local.NetworkID,
		ExpectedRole: expectedRole,
		Peers:        peer.NewStore(paths),
		Revocations:  revocation.NewStore(paths),
		Now:          time.Now,
	}}
	tlsConfig, err := tlsidentity.NewTLSConfig(local, verifier, time.Now().UTC())
	if err != nil {
		return nil, identity.Identity{}, err
	}
	return tlsConfig, local, nil
}

type wireMessage struct {
	Type      string             `json:"type"`
	NodeID    identity.NodeID    `json:"node_id"`
	NetworkID identity.NetworkID `json:"network_id"`
	Role      identity.Role      `json:"role"`
	Time      time.Time          `json:"time"`
}

func readMessage(reader io.Reader) (wireMessage, error) {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return wireMessage{}, err
	}
	length := binary.BigEndian.Uint32(header[:])
	if length == 0 || length > MaxFrameSize {
		return wireMessage{}, fmt.Errorf("%w: %d", ErrFrameTooLarge, length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return wireMessage{}, err
	}
	var message wireMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return wireMessage{}, fmt.Errorf("%w: decode: %w", ErrInvalidMessage, err)
	}
	if message.Type == "" {
		return wireMessage{}, fmt.Errorf("%w: missing type", ErrInvalidMessage)
	}
	return message, nil
}

func writeMessage(writer io.Writer, message wireMessage) error {
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
