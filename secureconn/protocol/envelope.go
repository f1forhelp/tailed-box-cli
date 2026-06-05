package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	Magic           = "TBXM"
	Version   uint8 = 1
	HeaderLen       = 36

	MaxPayloadSize = 1 << 20
)

type PacketType uint8

const (
	PacketTypeClientHello     PacketType = 1
	PacketTypeServerHello     PacketType = 2
	PacketTypeClientAuth      PacketType = 3
	PacketTypeEncryptedData   PacketType = 4
	PacketTypeRekey           PacketType = 5
	PacketTypeClose           PacketType = 6
	PacketTypeEnrollRequest   PacketType = 7
	PacketTypeEnrollChallenge PacketType = 8
	PacketTypeEnrollProof     PacketType = 9
	PacketTypeEnrollAccept    PacketType = 10
	PacketTypeEnrollReject    PacketType = 11
)

type SessionID [16]byte

type Packet struct {
	Type      PacketType
	Flags     uint16
	SessionID SessionID
	Sequence  uint64
	Payload   []byte
}

func Encode(packet Packet) ([]byte, error) {
	if !validPacketType(packet.Type) {
		return nil, fmt.Errorf("unsupported mesh packet type %d", packet.Type)
	}
	if packet.Flags != 0 {
		return nil, errors.New("mesh packet flags are reserved for future use")
	}
	if len(packet.Payload) > MaxPayloadSize {
		return nil, fmt.Errorf("mesh packet payload exceeds %d bytes", MaxPayloadSize)
	}

	data := make([]byte, HeaderLen+len(packet.Payload))
	copy(data[0:4], Magic)
	data[4] = Version
	data[5] = byte(packet.Type)
	binary.BigEndian.PutUint16(data[6:8], packet.Flags)
	copy(data[8:24], packet.SessionID[:])
	binary.BigEndian.PutUint64(data[24:32], packet.Sequence)
	binary.BigEndian.PutUint32(data[32:36], uint32(len(packet.Payload)))
	copy(data[HeaderLen:], packet.Payload)
	return data, nil
}

func Decode(data []byte) (Packet, error) {
	if len(data) < HeaderLen {
		return Packet{}, fmt.Errorf("mesh packet too short: got %d bytes", len(data))
	}
	if string(data[0:4]) != Magic {
		return Packet{}, errors.New("mesh packet magic mismatch")
	}
	if data[4] != Version {
		return Packet{}, fmt.Errorf("unsupported mesh packet version %d", data[4])
	}

	packetType := PacketType(data[5])
	if !validPacketType(packetType) {
		return Packet{}, fmt.Errorf("unsupported mesh packet type %d", packetType)
	}

	flags := binary.BigEndian.Uint16(data[6:8])
	if flags != 0 {
		return Packet{}, errors.New("mesh packet flags are reserved for future use")
	}

	payloadLen := binary.BigEndian.Uint32(data[32:36])
	if payloadLen > MaxPayloadSize {
		return Packet{}, fmt.Errorf("mesh packet payload exceeds %d bytes", MaxPayloadSize)
	}
	if len(data)-HeaderLen != int(payloadLen) {
		return Packet{}, fmt.Errorf("mesh packet payload length mismatch: header=%d actual=%d", payloadLen, len(data)-HeaderLen)
	}

	var sessionID SessionID
	copy(sessionID[:], data[8:24])
	payload := make([]byte, payloadLen)
	copy(payload, data[HeaderLen:])

	return Packet{
		Type:      packetType,
		Flags:     flags,
		SessionID: sessionID,
		Sequence:  binary.BigEndian.Uint64(data[24:32]),
		Payload:   payload,
	}, nil
}

func validPacketType(packetType PacketType) bool {
	switch packetType {
	case PacketTypeClientHello,
		PacketTypeServerHello,
		PacketTypeClientAuth,
		PacketTypeEncryptedData,
		PacketTypeRekey,
		PacketTypeClose,
		PacketTypeEnrollRequest,
		PacketTypeEnrollProof,
		PacketTypeEnrollAccept,
		PacketTypeEnrollReject,
		PacketTypeEnrollChallenge:
		return true
	default:
		return false
	}
}
