package auth

// Validator can be used to get the namespace identifier for a given username and password.
// In some cases this would be the username, but if the auth provider has a notion if projects
// users would probably want to use the same terraform states with different username / password
// combinations. The auth provider could return the project id instead to make this possible.
type Validator interface {
	Validate(username string, password string) (string, error)
}
