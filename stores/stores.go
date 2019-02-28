package stores

import (
	"errors"
	"sync"
)

// ErrorNotFound ...
var ErrorNotFound = errors.New("Not Found")

// ErrorLockedConflict ...
var ErrorLockedConflict = errors.New("Locked Conflict")

// ErrorNotImplemented ...
var ErrorNotImplemented = errors.New("Not Yet Implemented")

// TerraformState ...
type TerraformState struct {
	Identifier string
	IsLocked   bool
	LockID     string
	Contents   string
}

// NewTerraformState ...
func NewTerraformState(identifier string, contents string) *TerraformState {
	return &TerraformState{
		Identifier: identifier,
		IsLocked:   false,
		LockID:     "",
		Contents:   contents,
	}
}

// TerraformStateStore ...
type TerraformStateStore interface {
	ValidateAuth(username string, password string) error
	RetrieveState(username string, password string, identifier string) (*TerraformState, error)
	UpdateState(username string, password string, identifier string, contents string, lockID string) error
	LockState(username string, password string, identifier string, lockID string) (string, error)
	UnlockState(username string, password string, identifier string, lockID string) error
	DeleteState(username string, password string, identifier string) error
}

// InMemoryTerraformStateStore can be used for testing purposes
type InMemoryTerraformStateStore struct {
	items []*TerraformState
	mutex *sync.Mutex
}

// NewInMemoryTerraformStateStore ...
func NewInMemoryTerraformStateStore() *InMemoryTerraformStateStore {
	return &InMemoryTerraformStateStore{
		items: []*TerraformState{},
		mutex: &sync.Mutex{},
	}
}

// ValidateAuth ...
func (s *InMemoryTerraformStateStore) ValidateAuth(username string, password string) error {
	return nil
}

// RetrieveState ...
func (s *InMemoryTerraformStateStore) RetrieveState(username string, password string, identifier string) (*TerraformState, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, element := range s.items {
		if element.Identifier == identifier {
			return element, nil
		}
	}

	return nil, ErrorNotFound
}

// UpdateState ...
func (s *InMemoryTerraformStateStore) UpdateState(username string, password string, identifier string, contents string, lockID string) error {
	state, error := s.RetrieveState(username, password, identifier)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if error == ErrorNotFound {
		s.items = append(s.items, NewTerraformState(identifier, contents))
		return nil
	}

	if error != nil {
		return error
	}

	if state.IsLocked && lockID != state.LockID {
		return ErrorLockedConflict
	}

	state.Contents = contents

	return nil
}

// LockState ...
func (s *InMemoryTerraformStateStore) LockState(username string, password string, identifier string, lockID string) (string, error) {
	state, error := s.RetrieveState(username, password, identifier)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if error == ErrorNotFound {
		state = NewTerraformState(identifier, "")
		s.items = append(s.items, state)
	} else if error != nil {
		return "", error
	}

	if state.IsLocked {
		return state.LockID, ErrorLockedConflict
	}

	state.IsLocked = true
	state.LockID = lockID

	return "", nil
}

// UnlockState ...
func (s *InMemoryTerraformStateStore) UnlockState(username string, password string, identifier string, lockID string) error {
	state, error := s.RetrieveState(username, password, identifier)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if error != nil {
		return error
	}

	if !state.IsLocked {
		return nil
	}

	if lockID != "" && lockID != state.LockID {
		return ErrorLockedConflict
	}

	state.IsLocked = false
	state.LockID = ""

	return nil
}

// DeleteState ...
func (s *InMemoryTerraformStateStore) DeleteState(username string, password string, identifier string) error {
	return ErrorNotImplemented
}
