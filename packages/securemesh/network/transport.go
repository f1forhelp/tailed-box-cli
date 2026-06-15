package network

import (
	"context"
	"net/netip"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

type Endpoint struct {
	Address  netip.AddrPort `json:"address"`
	Protocol Protocol       `json:"protocol"`
}

type DialOptions struct {
	NetworkID      identity.NetworkID `json:"network_id"`
	LocalNodeID    identity.NodeID    `json:"local_node_id"`
	RemoteNodeID   identity.NodeID    `json:"remote_node_id"`
	ExpectedRole   identity.Role      `json:"expected_role"`
	RemoteEndpoint Endpoint           `json:"remote_endpoint"`
}

type ListenOptions struct {
	NetworkID   identity.NetworkID `json:"network_id"`
	LocalNodeID identity.NodeID    `json:"local_node_id"`
	Bind        Endpoint           `json:"bind"`
}

type Message struct {
	Type    MessageType `json:"type"`
	Payload []byte      `json:"payload"`
}

type PeerAuthenticator interface {
	AuthenticatePeer(ctx context.Context, metadata SessionMetadata) error
}

type Session interface {
	Metadata() SessionMetadata
	Send(ctx context.Context, message Message) error
	Receive(ctx context.Context) (Message, error)
	Close() error
}

type Transport interface {
	Listen(ctx context.Context, options ListenOptions) error
	Dial(ctx context.Context, options DialOptions) (Session, error)
	Close() error
}
