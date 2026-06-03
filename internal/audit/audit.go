package audit

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tailedbox/tailedbox/internal/secrets"
)

const (
	ActionJoinCodeCreated = "join_code.created"
	ActionJoinAttempt     = "join.attempt"
	ActionJoinSucceeded   = "join.succeeded"
	ActionJoinFailed      = "join.failed"
)

type Event struct {
	Version      int               `json:"version"`
	ID           string            `json:"id"`
	Time         time.Time         `json:"time"`
	Action       string            `json:"action"`
	ActorNodeID  string            `json:"actor_node_id,omitempty"`
	TargetNodeID string            `json:"target_node_id,omitempty"`
	Role         string            `json:"role,omitempty"`
	JoinCodeID   string            `json:"join_code_id,omitempty"`
	Result       string            `json:"result"`
	Reason       string            `json:"reason,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func Append(path string, event Event) error {
	if path == "" {
		return errors.New("audit log path is empty")
	}
	if event.Version == 0 {
		event.Version = 1
	}
	if event.ID == "" {
		id, err := newEventID()
		if err != nil {
			return err
		}
		event.ID = id
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if err := secrets.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, secrets.PrivateFileMode)
	if err != nil {
		return fmt.Errorf("open audit log %q: %w", path, err)
	}
	defer file.Close()
	if err := file.Chmod(secrets.PrivateFileMode); err != nil {
		return fmt.Errorf("secure audit log %q: %w", path, err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}
	data = append(data, '\n')
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func newEventID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate audit event id: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return "evt_" + strings.ToLower(encoded), nil
}
