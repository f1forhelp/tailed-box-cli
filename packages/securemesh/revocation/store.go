package revocation

import (
	"errors"
	"os"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

const stateVersion = 1

var ErrRevocationNotFound = errors.New("revocation not found")

type Store struct {
	paths config.Paths
	now   func() time.Time
}

func NewStore(paths config.Paths) Store {
	return Store{paths: paths, now: time.Now}
}

func (s Store) WithClock(now func() time.Time) Store {
	if now != nil {
		s.now = now
	}
	return s
}

func NewRecord(nodeID identity.NodeID, role identity.Role, revokedBy identity.NodeID, reason Reason, revokedAt time.Time) (Record, error) {
	record := Record{
		NodeID:    nodeID,
		Role:      role,
		RevokedAt: normalizeTime(revokedAt),
		RevokedBy: revokedBy,
		Reason:    reason,
	}
	if err := record.Validate(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s Store) Revoke(nodeID identity.NodeID, role identity.Role, revokedBy identity.NodeID, reason Reason) (Record, error) {
	lock, err := s.lock()
	if err != nil {
		return Record{}, err
	}
	defer lock.Release()

	state, err := s.load()
	if err != nil {
		return Record{}, err
	}
	for _, existing := range state.Records {
		if existing.NodeID == nodeID {
			return existing, nil
		}
	}

	record, err := NewRecord(nodeID, role, revokedBy, reason, s.nowUTC())
	if err != nil {
		return Record{}, err
	}
	state.Records = append(state.Records, record)
	if err := s.save(state); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s Store) IsRevoked(nodeID identity.NodeID) (bool, error) {
	if err := nodeID.Validate(); err != nil {
		return false, err
	}
	state, err := s.load()
	if err != nil {
		return false, err
	}
	for _, record := range state.Records {
		if record.NodeID == nodeID {
			return true, nil
		}
	}
	return false, nil
}

func (s Store) Get(nodeID identity.NodeID) (Record, error) {
	if err := nodeID.Validate(); err != nil {
		return Record{}, err
	}
	state, err := s.load()
	if err != nil {
		return Record{}, err
	}
	for _, record := range state.Records {
		if record.NodeID == nodeID {
			return record, nil
		}
	}
	return Record{}, ErrRevocationNotFound
}

func (s Store) List() ([]Record, error) {
	state, err := s.load()
	if err != nil {
		return nil, err
	}
	records := make([]Record, len(state.Records))
	copy(records, state.Records)
	return records, nil
}

func (s Store) lock() (*config.DirLock, error) {
	if err := s.paths.Ensure(); err != nil {
		return nil, err
	}
	return config.AcquireLock(s.paths.LockPath("revocations"))
}

func (s Store) load() (storeState, error) {
	var state storeState
	if err := config.LoadJSON(s.paths.RevocationsPath(), &state); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storeState{Version: stateVersion}, nil
		}
		return storeState{}, err
	}
	if state.Version == 0 {
		state.Version = stateVersion
	}
	for _, record := range state.Records {
		if err := record.Validate(); err != nil {
			return storeState{}, err
		}
	}
	return state, nil
}

func (s Store) save(state storeState) error {
	state.Version = stateVersion
	return config.SaveJSON(s.paths.RevocationsPath(), state, config.FileMode)
}

func (s Store) nowUTC() time.Time {
	now := s.now
	if now == nil {
		now = time.Now
	}
	return now().UTC()
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

type storeState struct {
	Version int      `json:"version"`
	Records []Record `json:"records"`
}
