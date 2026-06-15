package network

import (
	"errors"
	"fmt"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

var ErrInvalidSession = errors.New("invalid network session")

type Protocol string

const (
	ProtocolUDP  Protocol = "udp"
	ProtocolQUIC Protocol = "quic"
)

func (p Protocol) Valid() bool {
	switch p {
	case ProtocolUDP, ProtocolQUIC:
		return true
	default:
		return false
	}
}

func (p Protocol) Validate() error {
	if !p.Valid() {
		return fmt.Errorf("invalid protocol %q", p)
	}
	return nil
}

type SessionID string

func (id SessionID) String() string {
	return string(id)
}

func (id SessionID) Validate() error {
	if id == "" {
		return fmt.Errorf("%w: empty session id", ErrInvalidSession)
	}
	return nil
}

type SessionState string

const (
	SessionStateHandshaking SessionState = "handshaking"
	SessionStateEstablished SessionState = "established"
	SessionStateClosed      SessionState = "closed"
)

func (s SessionState) Valid() bool {
	switch s {
	case SessionStateHandshaking, SessionStateEstablished, SessionStateClosed:
		return true
	default:
		return false
	}
}

func (s SessionState) Validate() error {
	if !s.Valid() {
		return fmt.Errorf("%w: invalid state %q", ErrInvalidSession, s)
	}
	return nil
}

type PacketType uint8

const (
	PacketTypeHandshake PacketType = 1
	PacketTypeData      PacketType = 2
	PacketTypeControl   PacketType = 3
)

type MessageType uint16

const (
	MessageTypeControl MessageType = 1
	MessageTypeData    MessageType = 2
)

type SessionMetadata struct {
	ID            SessionID          `json:"id"`
	NetworkID     identity.NetworkID `json:"network_id"`
	LocalNodeID   identity.NodeID    `json:"local_node_id"`
	RemoteNodeID  identity.NodeID    `json:"remote_node_id"`
	RemoteRole    identity.Role      `json:"remote_role"`
	Protocol      Protocol           `json:"protocol"`
	State         SessionState       `json:"state"`
	KeyEpoch      uint64             `json:"key_epoch"`
	EstablishedAt time.Time          `json:"established_at,omitempty"`
}

func (m SessionMetadata) Validate() error {
	if err := m.ID.Validate(); err != nil {
		return err
	}
	if err := m.NetworkID.Validate(); err != nil {
		return err
	}
	if err := m.LocalNodeID.Validate(); err != nil {
		return err
	}
	if err := m.RemoteNodeID.Validate(); err != nil {
		return err
	}
	if err := m.RemoteRole.Validate(); err != nil {
		return err
	}
	if err := m.Protocol.Validate(); err != nil {
		return err
	}
	if err := m.State.Validate(); err != nil {
		return err
	}
	return nil
}
