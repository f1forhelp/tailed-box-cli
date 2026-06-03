package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type MessageType string

const (
	MessageTypePing             MessageType = "ping"
	MessageTypePong             MessageType = "pong"
	MessageTypePeerUpdate       MessageType = "peer_update"
	MessageTypeStatusRequest    MessageType = "status_request"
	MessageTypeStatusResponse   MessageType = "status_response"
	MessageTypeDiagnoseRequest  MessageType = "diagnose_request"
	MessageTypeDiagnoseResponse MessageType = "diagnose_response"
	MessageTypeEnrollRequest    MessageType = "enroll_request"
	MessageTypeEnrollAccept     MessageType = "enroll_accept"
	MessageTypeEnrollReject     MessageType = "enroll_reject"
)

type ControlMessage struct {
	Version    int             `json:"version"`
	Type       MessageType     `json:"type"`
	ID         string          `json:"id,omitempty"`
	NodeID     string          `json:"node_id,omitempty"`
	PeerNodeID string          `json:"peer_node_id,omitempty"`
	SentAt     time.Time       `json:"sent_at,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

func EncodeControlMessage(message ControlMessage) ([]byte, error) {
	if message.Version == 0 {
		message.Version = 1
	}
	if message.Type == "" {
		return nil, errors.New("mesh control message type is required")
	}
	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal mesh control message: %w", err)
	}
	return data, nil
}

func DecodeControlMessage(data []byte) (ControlMessage, error) {
	var message ControlMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return ControlMessage{}, fmt.Errorf("parse mesh control message: %w", err)
	}
	if message.Version != 1 {
		return ControlMessage{}, fmt.Errorf("unsupported mesh control message version %d", message.Version)
	}
	if message.Type == "" {
		return ControlMessage{}, errors.New("mesh control message type is required")
	}
	return message, nil
}
