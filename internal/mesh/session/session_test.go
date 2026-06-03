package session

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	meshcrypto "github.com/tailedbox/tailedbox/internal/mesh/crypto"
	"github.com/tailedbox/tailedbox/internal/mesh/protocol"
)

func TestReplayWindowRejectsDuplicatesAndStalePackets(t *testing.T) {
	window := NewReplayWindow(4)
	for _, sequence := range []uint64{1, 2, 5, 3} {
		if !window.Accept(sequence) {
			t.Fatalf("expected sequence %d to be accepted", sequence)
		}
	}
	for _, sequence := range []uint64{0, 2, 1} {
		if window.Accept(sequence) {
			t.Fatalf("expected sequence %d to be rejected", sequence)
		}
	}
}

func TestSenderReceiverSealOpenRoundTrip(t *testing.T) {
	sender, receiver := testSenderReceiver(t)
	packet, err := sender.Seal(protocol.PacketTypeEncryptedData, []byte("ping"))
	if err != nil {
		t.Fatalf("seal packet: %v", err)
	}
	if packet.Sequence != 1 {
		t.Fatalf("sequence = %d, want 1", packet.Sequence)
	}
	plaintext, err := receiver.Open(packet)
	if err != nil {
		t.Fatalf("open packet: %v", err)
	}
	if !bytes.Equal(plaintext, []byte("ping")) {
		t.Fatalf("plaintext = %q", plaintext)
	}
}

func TestReceiverRejectsReplay(t *testing.T) {
	sender, receiver := testSenderReceiver(t)
	packet, err := sender.Seal(protocol.PacketTypeEncryptedData, []byte("ping"))
	if err != nil {
		t.Fatalf("seal packet: %v", err)
	}
	if _, err := receiver.Open(packet); err != nil {
		t.Fatalf("first open: %v", err)
	}
	if _, err := receiver.Open(packet); !errors.Is(err, ErrReplayRejected) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}

func TestPacketHeaderIsAssociatedData(t *testing.T) {
	sender, receiver := testSenderReceiver(t)
	packet, err := sender.Seal(protocol.PacketTypeEncryptedData, []byte("ping"))
	if err != nil {
		t.Fatalf("seal packet: %v", err)
	}

	tamperedType := packet
	tamperedType.Type = protocol.PacketTypeClose
	if _, err := receiver.Open(tamperedType); err == nil || !strings.Contains(err.Error(), "open encrypted mesh packet") {
		t.Fatalf("expected packet type tamper to fail authentication, got %v", err)
	}

	tamperedSequence := packet
	tamperedSequence.Sequence++
	if _, err := receiver.Open(tamperedSequence); err == nil || !strings.Contains(err.Error(), "open encrypted mesh packet") {
		t.Fatalf("expected sequence tamper to fail authentication, got %v", err)
	}
}

func TestSenderRejectsHandshakePacketTypes(t *testing.T) {
	sender, _ := testSenderReceiver(t)
	if _, err := sender.Seal(protocol.PacketTypeClientHello, []byte("hello")); !errors.Is(err, ErrInvalidPacket) {
		t.Fatalf("expected invalid packet error, got %v", err)
	}
}

func testSenderReceiver(t *testing.T) (*Sender, *Receiver) {
	t.Helper()
	var sessionID protocol.SessionID
	copy(sessionID[:], []byte("0123456789abcdef"))
	key := bytes.Repeat([]byte{7}, meshcrypto.SessionKeySize)
	noncePrefix := []byte{1, 2, 3, 4}

	sender, err := NewSender(sessionID, key, noncePrefix)
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}
	receiver, err := NewReceiver(sessionID, key, noncePrefix, NewReplayWindow(8))
	if err != nil {
		t.Fatalf("new receiver: %v", err)
	}
	return sender, receiver
}
