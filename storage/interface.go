package storage

import (
	"errors"
)

// ErrorNotFound will be returned when object with given identifier does not exist in storage
var ErrorNotFound = errors.New("Not Found")

// ErrorLockedConflict will be returned when object is locked by someone else
var ErrorLockedConflict = errors.New("Locked Conflict")

// ErrorNotImplemented will be returned if the store did not yet implement a certain method in StateStore interface
var ErrorNotImplemented = errors.New("Not Yet Implemented")

// TerraformState holds all the information necessary for Terraform
type TerraformState struct {
	Namespace  string
	Identifier string
	IsLocked   bool
	LockID     string
	Contents   string
}

// NewTerraformState returns a fresh TerraformState instance
func NewTerraformState(namespace string, identifier string, contents string) *TerraformState {
	return &TerraformState{
		Namespace:  namespace,
		Identifier: identifier,
		IsLocked:   false,
		LockID:     "",
		Contents:   contents,
	}
}

// StateStore can be used to get, lock, delete state using different backend implementations
// namespace is returned by auth.Provider interface for a given username and password combination
type StateStore interface {
	Get(namespace string, identifier string) (*TerraformState, error)
	Update(namespace string, identifier string, contents string, lockID string) error
	// Lock returns empty values in case of success. In case of lock conflict, it returns the current lock holder ID
	Lock(namespace string, identifier string, lockID string) (string, error)
	// Unlock only succeeds with empty lockID or with the correct lockID
	Unlock(namespace string, identifier string, lockID string) error
	ForceUnlock(namespace string, identifier string) error
	Delete(namespace string, identifier string) error
}
