//go:build windows

package auth

import (
	"log"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modadvapi32          = windows.NewLazySystemDLL("advapi32.dll")
	procLogonUserW       = modadvapi32.NewProc("LogonUserW")
)

func logonUser(username, domain, password string, logonType, logonProvider uint32) (windows.Token, error) {
	u, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return 0, err
	}
	d, err := windows.UTF16PtrFromString(domain)
	if err != nil {
		return 0, err
	}
	p, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return 0, err
	}

	var token windows.Token
	ret, _, err := procLogonUserW.Call(
		uintptr(unsafe.Pointer(u)),
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(p)),
		uintptr(logonType),
		uintptr(logonProvider),
		uintptr(unsafe.Pointer(&token)),
	)
	if ret == 0 {
		return 0, err
	}
	return token, nil
}

func (a *Authenticator) verifyOSUser(username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	domain := ""
	if idx := strings.Index(username, "\\"); idx != -1 {
		domain = username[:idx]
		username = username[idx+1:]
	} else if strings.Contains(username, "@") {
	} else {
		domain = "."
	}

	const (
		LOGON32_LOGON_NETWORK    = 3
		LOGON32_PROVIDER_DEFAULT = 0
	)

	token, err := logonUser(username, domain, password, LOGON32_LOGON_NETWORK, LOGON32_PROVIDER_DEFAULT)
	if err != nil {
		log.Printf("OS auth: Windows credential verification failed for user '%s': %v", username, err)
		return false
	}
	token.Close()

	log.Printf("OS auth: Windows user '%s' authenticated successfully", username)
	return true
}
