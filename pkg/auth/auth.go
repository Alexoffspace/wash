package auth

import (
	"fmt"
)

// Credentials represents authorization credentials
type Credentials struct {
	Token    string
	Login    string
	Password string
}

// Authenticator is responsible for credential verification
type Authenticator struct {
	tokens       []string
	enableOSAuth bool
}

// NewAuthenticator creates an authenticator
func NewAuthenticator(tokens []string, enableOSAuth bool) *Authenticator {
	return &Authenticator{
		tokens:       tokens,
		enableOSAuth: enableOSAuth,
	}
}

// Authenticate checks credentials
func (a *Authenticator) Authenticate(user string, password string) (string, error) {
	// First check token
	if user == "" && password != "" {
		// Password is used as token
		for _, t := range a.tokens {
			if password == t {
				return "token-user", nil
			}
		}
		return "", fmt.Errorf("invalid token")
	}

	// Check login/password
	if user != "" && password != "" {
		// First check token-login
		for _, t := range a.tokens {
			if user == t {
				return "token-user", nil
			}
		}

		// Check token-password
		for _, t := range a.tokens {
			if password == t {
				return "token-user", nil
			}
		}

		// Check OS authentication
		if a.enableOSAuth {
			if a.verifyOSUser(user, password) {
				return user, nil
			}
		}

		return "", fmt.Errorf("invalid credentials")
	}

	return "", fmt.Errorf("empty credentials")
}

// HasTokenAuth returns true if tokens are loaded
func (a *Authenticator) HasTokenAuth() bool {
	return len(a.tokens) > 0
}

// HasOSAuth returns true if OS authentication is enabled
func (a *Authenticator) HasOSAuth() bool {
	return a.enableOSAuth
}
