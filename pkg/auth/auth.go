package auth

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
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

// verifyOSUser checks the user via the OS
func (a *Authenticator) verifyOSUser(username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: verify password using runas (requires elevation) or PowerShell
		// Use PowerShell to validate credentials
		// Note: This requires the user to be in the system, and uses DirectoryServices
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf(`
$username = '%s'
$password = '%s'
$domain = $env:USERDOMAIN
try {
    $cred = New-Object System.Management.Automation.PSCredential("$domain\$username", (ConvertTo-SecureString $password -AsPlainText -Force))
    $null = New-Object DirectoryServices.DirectoryEntry("", "$domain\$username", $password)
    exit 0
} catch {
    exit 1
}
`, escapeWindowsString(username), escapeWindowsString(password)))
		err := cmd.Run()
		if err != nil {
			log.Printf("OS auth: Windows credential verification failed for user '%s'", username)
			return false
		}
		log.Printf("OS auth: Windows user '%s' authenticated successfully", username)
		return true
	}

	// Linux: verify password via su
	cmd = exec.Command("su", "-c", "echo ok", username)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false
	}
	stdin.Write([]byte(password + "\n"))
	stdin.Close()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "ok")
}

// escapeWindowsString escapes special characters for Windows PowerShell
func escapeWindowsString(s string) string {
	// Escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "''")
}

// HasTokenAuth returns true if tokens are loaded
func (a *Authenticator) HasTokenAuth() bool {
	return len(a.tokens) > 0
}

// HasOSAuth returns true if OS authentication is enabled
func (a *Authenticator) HasOSAuth() bool {
	return a.enableOSAuth
}
