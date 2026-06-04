package protocol

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

func TestPacketEncodeDecodeRoundTrip(t *testing.T) {
	var session SessionID
	copy(session[:], []byte("0123456789abcdef"))

	encoded, err := Encode(Packet{
		Type:      PacketTypeEncryptedData,
		SessionID: session,
		Sequence:  42,
		Payload:   []byte("ciphertext"),
	})
	if err != nil {
		t.Fatalf("encode packet: %v", err)
	}
	if len(encoded) != HeaderLen+len("ciphertext") {
		t.Fatalf("encoded length = %d, want %d", len(encoded), HeaderLen+len("ciphertext"))
	}
	if string(encoded[0:4]) != Magic {
		t.Fatalf("magic = %q, want %q", encoded[0:4], Magic)
	}
	if encoded[4] != Version {
		t.Fatalf("version = %d, want %d", encoded[4], Version)
	}
	if binary.BigEndian.Uint64(encoded[24:32]) != 42 {
		t.Fatalf("sequence not encoded in network byte order")
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode packet: %v", err)
	}
	if decoded.Type != PacketTypeEncryptedData {
		t.Fatalf("type = %d, want %d", decoded.Type, PacketTypeEncryptedData)
	}
	if decoded.Sequence != 42 {
		t.Fatalf("sequence = %d, want 42", decoded.Sequence)
	}
	if decoded.SessionID != session {
		t.Fatalf("session id mismatch")
	}
	if !bytes.Equal(decoded.Payload, []byte("ciphertext")) {
		t.Fatalf("payload = %q", decoded.Payload)
	}
}

func TestDecodeRejectsMalformedPackets(t *testing.T) {
	packet, err := Encode(Packet{Type: PacketTypeClientHello, Payload: []byte("hello")})
	if err != nil {
		t.Fatalf("encode packet: %v", err)
	}

	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "short",
			data: packet[:HeaderLen-1],
			want: "too short",
		},
		{
			name: "bad magic",
			data: withByte(packet, 0, 'X'),
			want: "magic",
		},
		{
			name: "bad version",
			data: withByte(packet, 4, 99),
			want: "version",
		},
		{
			name: "bad type",
			data: withByte(packet, 5, 99),
			want: "type",
		},
		{
			name: "reserved flags",
			data: withUint16(packet, 6, 1),
			want: "flags",
		},
		{
			name: "length mismatch",
			data: withUint32(packet, 32, 100),
			want: "length mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.data)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestControlMessageRoundTrip(t *testing.T) {
	sentAt := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	encoded, err := EncodeControlMessage(ControlMessage{
		Type:       MessageTypePing,
		ID:         "msg_1",
		NodeID:     "node_a",
		PeerNodeID: "node_b",
		SentAt:     sentAt,
		Payload:    []byte(`{"nonce":"abc"}`),
	})
	if err != nil {
		t.Fatalf("encode control message: %v", err)
	}

	decoded, err := DecodeControlMessage(encoded)
	if err != nil {
		t.Fatalf("decode control message: %v", err)
	}
	if decoded.Version != 1 {
		t.Fatalf("version = %d, want 1", decoded.Version)
	}
	if decoded.Type != MessageTypePing {
		t.Fatalf("type = %s, want %s", decoded.Type, MessageTypePing)
	}
	if decoded.NodeID != "node_a" || decoded.PeerNodeID != "node_b" {
		t.Fatalf("unexpected node ids: %#v", decoded)
	}
	if !decoded.SentAt.Equal(sentAt) {
		t.Fatalf("sent_at = %s, want %s", decoded.SentAt, sentAt)
	}
}

func withByte(data []byte, index int, value byte) []byte {
	clone := append([]byte(nil), data...)
	clone[index] = value
	return clone
}

func withUint16(data []byte, index int, value uint16) []byte {
	clone := append([]byte(nil), data...)
	binary.BigEndian.PutUint16(clone[index:index+2], value)
	return clone
}

func withUint32(data []byte, index int, value uint32) []byte {
	clone := append([]byte(nil), data...)
	binary.BigEndian.PutUint32(clone[index:index+4], value)
	return clone
}
