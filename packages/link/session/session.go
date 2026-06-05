package session

import (
	"crypto/cipher"
	"errors"
	"fmt"

	meshcrypto "github.com/tailedbox/link/crypto"
	"github.com/tailedbox/link/protocol"
)

const DefaultReplayWindowSize uint64 = 1024

var (
	ErrReplayRejected = errors.New("mesh replay rejected")
	ErrInvalidPacket  = errors.New("invalid encrypted mesh packet")
)

type ReplayWindow struct {
	size    uint64
	highest uint64
	seen    map[uint64]struct{}
}

func NewReplayWindow(size uint64) *ReplayWindow {
	if size == 0 {
		size = DefaultReplayWindowSize
	}
	return &ReplayWindow{
		size: size,
		seen: make(map[uint64]struct{}),
	}
}

func (w *ReplayWindow) Accept(sequence uint64) bool {
	if w == nil || sequence == 0 {
		return false
	}
	if w.highest != 0 && sequence <= w.highest && w.highest-sequence >= w.size {
		return false
	}
	if _, ok := w.seen[sequence]; ok {
		return false
	}
	if sequence > w.highest {
		w.highest = sequence
		w.prune()
	}
	w.seen[sequence] = struct{}{}
	return true
}

type Sender struct {
	sessionID    protocol.SessionID
	aead         cipher.AEAD
	noncePrefix  []byte
	nextSequence uint64
}

type Receiver struct {
	sessionID   protocol.SessionID
	aead        cipher.AEAD
	noncePrefix []byte
	replay      *ReplayWindow
}

func NewSender(sessionID protocol.SessionID, key, noncePrefix []byte) (*Sender, error) {
	aead, err := newPacketAEAD(sessionID, key, noncePrefix)
	if err != nil {
		return nil, err
	}
	return &Sender{
		sessionID:   sessionID,
		aead:        aead,
		noncePrefix: cloneBytes(noncePrefix),
	}, nil
}

func NewReceiver(sessionID protocol.SessionID, key, noncePrefix []byte, replay *ReplayWindow) (*Receiver, error) {
	aead, err := newPacketAEAD(sessionID, key, noncePrefix)
	if err != nil {
		return nil, err
	}
	if replay == nil {
		replay = NewReplayWindow(DefaultReplayWindowSize)
	}
	return &Receiver{
		sessionID:   sessionID,
		aead:        aead,
		noncePrefix: cloneBytes(noncePrefix),
		replay:      replay,
	}, nil
}

func (s *Sender) Seal(packetType protocol.PacketType, plaintext []byte) (protocol.Packet, error) {
	if s == nil {
		return protocol.Packet{}, errors.New("mesh session sender is nil")
	}
	if !encryptedPacketType(packetType) {
		return protocol.Packet{}, fmt.Errorf("%w: unsupported packet type %d", ErrInvalidPacket, packetType)
	}
	if s.nextSequence == ^uint64(0) {
		return protocol.Packet{}, errors.New("mesh session sequence exhausted")
	}
	sequence := s.nextSequence + 1
	packet := protocol.Packet{
		Type:      packetType,
		SessionID: s.sessionID,
		Sequence:  sequence,
		Payload:   make([]byte, len(plaintext)+s.aead.Overhead()),
	}
	associatedData, err := associatedData(packet)
	if err != nil {
		return protocol.Packet{}, err
	}
	nonce, err := meshcrypto.Nonce(s.noncePrefix, sequence)
	if err != nil {
		return protocol.Packet{}, err
	}
	packet.Payload = s.aead.Seal(nil, nonce, plaintext, associatedData)
	s.nextSequence = sequence
	return packet, nil
}

func (r *Receiver) Open(packet protocol.Packet) ([]byte, error) {
	if r == nil {
		return nil, errors.New("mesh session receiver is nil")
	}
	if !encryptedPacketType(packet.Type) {
		return nil, fmt.Errorf("%w: unsupported packet type %d", ErrInvalidPacket, packet.Type)
	}
	if packet.SessionID != r.sessionID {
		return nil, fmt.Errorf("%w: session id mismatch", ErrInvalidPacket)
	}
	if packet.Sequence == 0 {
		return nil, fmt.Errorf("%w: encrypted packet sequence is zero", ErrInvalidPacket)
	}
	associatedData, err := associatedData(packet)
	if err != nil {
		return nil, err
	}
	nonce, err := meshcrypto.Nonce(r.noncePrefix, packet.Sequence)
	if err != nil {
		return nil, err
	}
	plaintext, err := r.aead.Open(nil, nonce, packet.Payload, associatedData)
	if err != nil {
		return nil, fmt.Errorf("open encrypted mesh packet: %w", err)
	}
	if !r.replay.Accept(packet.Sequence) {
		return nil, fmt.Errorf("%w: sequence %d", ErrReplayRejected, packet.Sequence)
	}
	return plaintext, nil
}

func (w *ReplayWindow) prune() {
	if w.highest <= w.size {
		return
	}
	threshold := w.highest - w.size
	for sequence := range w.seen {
		if sequence <= threshold {
			delete(w.seen, sequence)
		}
	}
}

func newPacketAEAD(sessionID protocol.SessionID, key, noncePrefix []byte) (cipher.AEAD, error) {
	if zeroSessionID(sessionID) {
		return nil, fmt.Errorf("%w: session id is required", ErrInvalidPacket)
	}
	if len(noncePrefix) != meshcrypto.NoncePrefixLen {
		return nil, fmt.Errorf("%w: nonce prefix must be %d bytes", ErrInvalidPacket, meshcrypto.NoncePrefixLen)
	}
	aead, err := meshcrypto.NewAEAD(key)
	if err != nil {
		return nil, err
	}
	return aead, nil
}

func associatedData(packet protocol.Packet) ([]byte, error) {
	encoded, err := protocol.Encode(packet)
	if err != nil {
		return nil, err
	}
	return encoded[:protocol.HeaderLen], nil
}

func encryptedPacketType(packetType protocol.PacketType) bool {
	switch packetType {
	case protocol.PacketTypeClientAuth, protocol.PacketTypeEncryptedData, protocol.PacketTypeRekey, protocol.PacketTypeClose:
		return true
	default:
		return false
	}
}

func zeroSessionID(sessionID protocol.SessionID) bool {
	var zero protocol.SessionID
	return sessionID == zero
}

func cloneBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}
