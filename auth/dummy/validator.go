package dummy

import "github.com/stefreak/terraform-state-store/auth"

// NewValidator returns a validator that ignores the password. Username will be used as a namespace.
func NewValidator() auth.Validator {
	return &dummyValidator{}
}

type dummyValidator struct{}

func (v *dummyValidator) Validate(username string, password string) (string, error) {
	return username, nil
}
