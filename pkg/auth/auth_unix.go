//go:build !windows

package auth

import (
	"log"
	"os/exec"
	"strings"
)

func (a *Authenticator) verifyOSUser(username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	cmd := exec.Command("su", "-c", "echo ok", username)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("OS auth: failed to create stdin pipe for user '%s'", username)
		return false
	}
	_, err = stdin.Write([]byte(password + "\n"))
	if err != nil {
		log.Printf("OS auth: failed to write password for user '%s'", username)
		return false
	}
	stdin.Close()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("OS auth: su verification failed for user '%s'", username)
		return false
	}

	ok := strings.Contains(string(output), "ok")
	if ok {
		log.Printf("OS auth: user '%s' authenticated successfully", username)
	}
	return ok
}
