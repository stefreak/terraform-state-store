package inmemory

import "sync"
import "github.com/stefreak/terraform-state-store/storage"

// NewStateStore inmemory should only be used for testing purposes.
// It does not persist data across restarts, and it does not support namespaces or deleting right now.
func NewStateStore() storage.StateStore {
	return &inMemoryStateStore{
		items: []*storage.TerraformState{},
		mutex: &sync.Mutex{},
	}
}

type inMemoryStateStore struct {
	items []*storage.TerraformState
	mutex *sync.Mutex
}

// get does no locking on its own
func (s *inMemoryStateStore) get(namespace string, identifier string) (*storage.TerraformState, error) {
	for _, element := range s.items {
		if element.Identifier == identifier && element.Namespace == namespace {
			return element, nil
		}
	}

	return nil, storage.ErrorNotFound
}

func (s *inMemoryStateStore) Get(namespace string, identifier string) (*storage.TerraformState, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.get(namespace, identifier)
}

func (s *inMemoryStateStore) Update(namespace string, identifier string, contents string, lockID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	state, err := s.get(namespace, identifier)

	if err == storage.ErrorNotFound {
		s.items = append(s.items, storage.NewTerraformState(namespace, identifier, contents))
		return nil
	}

	if err != nil {
		return err
	}

	if state.IsLocked && lockID != state.LockID {
		return storage.ErrorLockedConflict
	}

	state.Contents = contents

	return nil
}

func (s *inMemoryStateStore) Lock(namespace string, identifier string, lockID string) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	state, error := s.get(namespace, identifier)

	if error != nil {
		return "", error
	}

	if state.IsLocked {
		return state.LockID, storage.ErrorLockedConflict
	}

	state.IsLocked = true
	state.LockID = lockID

	return "", nil
}

func (s *inMemoryStateStore) Unlock(namespace string, identifier string, lockID string) error {
	return s.unlock(namespace, identifier, lockID, false)
}

func (s *inMemoryStateStore) ForceUnlock(namespace string, identifier string) error {
	return s.unlock(namespace, identifier, "", true)
}

func (s *inMemoryStateStore) unlock(namespace string, identifier string, lockID string, force bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	state, err := s.get(namespace, identifier)

	if err != nil {
		return err
	}

	if !state.IsLocked {
		return nil
	}

	if !force && lockID != state.LockID {
		return storage.ErrorLockedConflict
	}

	state.IsLocked = false
	state.LockID = ""

	return nil
}

func (s *inMemoryStateStore) Delete(namespace string, identifier string) error {
	return storage.ErrorNotImplemented
}
