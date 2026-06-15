package join

import (
	"errors"
	"os"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
	securecrypto "github.com/f1forhelp/tailed-box-cli/packages/securemesh/crypto"
)

const stateVersion = 1

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

func (s Store) Create(request CreateRequest) (string, Record, error) {
	lock, err := s.lock()
	if err != nil {
		return "", Record{}, err
	}
	defer lock.Release()

	state, err := s.load()
	if err != nil {
		return "", Record{}, err
	}
	code, record, err := NewRecord(request, s.nowUTC())
	if err != nil {
		return "", Record{}, err
	}
	state.Records = append(state.Records, record)
	if err := s.save(state); err != nil {
		return "", Record{}, err
	}
	return code, record, nil
}

func (s Store) ValidateAndConsume(request ConsumeRequest) (ConsumeResult, error) {
	return s.ValidateAndConsumeWith(request, nil)
}

func (s Store) ValidateAndConsumeWith(request ConsumeRequest, beforeConsume func(Record) error) (ConsumeResult, error) {
	if err := request.Validate(); err != nil {
		return ConsumeResult{}, err
	}

	lock, err := s.lock()
	if err != nil {
		return ConsumeResult{}, err
	}
	defer lock.Release()

	state, err := s.load()
	if err != nil {
		return ConsumeResult{}, err
	}

	for idx := range state.Records {
		record := &state.Records[idx]
		if record.VerifierAlgorithm != VerifierAlgorithmV1 {
			continue
		}
		verifier, err := VerifierForCode(request.Code, record.Salt)
		if err != nil {
			return ConsumeResult{}, err
		}
		if !securecrypto.ConstantTimeEqual(verifier, record.Verifier) {
			continue
		}
		if record.Status == StatusConsumed {
			return ConsumeResult{}, ErrCodeConsumed
		}
		if !sameNetwork(record.NetworkID, request.NetworkID) {
			return ConsumeResult{}, ErrWrongNetwork
		}
		if record.ExpectedRole != request.ExpectedRole {
			return ConsumeResult{}, ErrWrongRole
		}
		if beforeConsume != nil {
			if err := beforeConsume(*record); err != nil {
				return ConsumeResult{}, err
			}
		}

		consumedAt := s.nowUTC()
		record.Status = StatusConsumed
		record.ConsumedAt = &consumedAt
		record.ConsumedBy = request.ConsumedBy
		if err := record.Validate(); err != nil {
			return ConsumeResult{}, err
		}
		if err := s.save(state); err != nil {
			return ConsumeResult{}, err
		}
		return ConsumeResult{Record: *record, Consumed: true}, nil
	}

	return ConsumeResult{}, ErrInvalidCode
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
	return config.AcquireLock(s.paths.LockPath("join-codes"))
}

func (s Store) load() (storeState, error) {
	var state storeState
	if err := config.LoadJSON(s.paths.JoinCodesPath(), &state); err != nil {
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
	return config.SaveJSON(s.paths.JoinCodesPath(), state, config.FileMode)
}

func (s Store) nowUTC() time.Time {
	now := s.now
	if now == nil {
		now = time.Now
	}
	return now().UTC()
}

type storeState struct {
	Version int      `json:"version"`
	Records []Record `json:"records"`
}
