package peer

import (
	"errors"
	"os"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/revocation"
)

const stateVersion = 1

var (
	ErrPeerNotFound = errors.New("peer not found")
	ErrPeerExists   = errors.New("peer already exists")
)

type Store struct {
	paths config.Paths
}

func NewStore(paths config.Paths) Store {
	return Store{paths: paths}
}

func (s Store) Add(record Record) error {
	if err := record.Validate(); err != nil {
		return err
	}
	lock, err := s.lock()
	if err != nil {
		return err
	}
	defer lock.Release()

	state, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range state.Records {
		if existing.NodeID == record.NodeID {
			return ErrPeerExists
		}
	}
	state.Records = append(state.Records, record)
	return s.save(state)
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
	return Record{}, ErrPeerNotFound
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

func (s Store) ActivePeers() ([]Record, error) {
	records, err := s.List()
	if err != nil {
		return nil, err
	}
	active := make([]Record, 0, len(records))
	for _, record := range records {
		if record.Active() {
			active = append(active, record)
		}
	}
	return active, nil
}

func (s Store) MarkRevoked(revocationRecord revocation.Record) error {
	if err := revocationRecord.Validate(); err != nil {
		return err
	}
	lock, err := s.lock()
	if err != nil {
		return err
	}
	defer lock.Release()

	state, err := s.load()
	if err != nil {
		return err
	}
	for idx := range state.Records {
		if state.Records[idx].NodeID == revocationRecord.NodeID {
			state.Records[idx].Status = StatusRevoked
			state.Records[idx].RevokedAt = &revocationRecord.RevokedAt
			return s.save(state)
		}
	}
	return ErrPeerNotFound
}

func (s Store) lock() (*config.DirLock, error) {
	if err := s.paths.Ensure(); err != nil {
		return nil, err
	}
	return config.AcquireLock(s.paths.LockPath("peers"))
}

func (s Store) load() (storeState, error) {
	var state storeState
	if err := config.LoadJSON(s.paths.PeersPath(), &state); err != nil {
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
	return config.SaveJSON(s.paths.PeersPath(), state, config.FileMode)
}

type storeState struct {
	Version int      `json:"version"`
	Records []Record `json:"records"`
}
