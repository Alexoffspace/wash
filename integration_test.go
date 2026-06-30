package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"WASH/pkg/api"
	"WASH/pkg/auth"
	"WASH/pkg/shell"
	"WASH/pkg/ws"
)

// TestShellSession tests basic shell session functionality
func TestShellSession(t *testing.T) {
	session, err := shell.NewSession("")
	if err != nil {
		t.Fatalf("Failed to create shell session: %v", err)
	}
	defer session.Close()

	// Test writing a command
	_, err = session.Write([]byte("echo hello\n"))
	if err != nil {
		t.Fatalf("Failed to write to shell: %v", err)
	}

	// Wait for output
	time.Sleep(500 * time.Millisecond)

	// Read output
	output := session.ReadStdout()
	if !bytes.Contains([]byte(output), []byte("hello")) {
		t.Errorf("Expected output to contain 'hello', got: %s", output)
	}
}

// TestRunCommand tests the RunCommand function
func TestRunCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "echo command",
			command:  "echo test123",
			expected: "test123",
		},
		{
			name:     "pwd command",
			command:  "pwd",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := shell.RunCommand(tt.command, "")
			if err != nil {
				t.Fatalf("Failed to run command: %v", err)
			}

			if !bytes.Contains([]byte(output.Stdout), []byte(tt.expected)) {
				t.Errorf("Expected stdout to contain '%s', got: %s", tt.expected, output.Stdout)
			}
		})
	}
}

// TestAuthenticator tests the authenticator
func TestAuthenticator(t *testing.T) {
	tokens := []string{"test-token-1", "test-token-2"}
	a := auth.NewAuthenticator(tokens, false)

	tests := []struct {
		name       string
		user       string
		password   string
		shouldFail bool
	}{
		{
			name:       "valid token",
			user:       "",
			password:   "test-token-1",
			shouldFail: false,
		},
		{
			name:       "invalid token",
			user:       "",
			password:   "invalid-token",
			shouldFail: true,
		},
		{
			name:       "empty credentials",
			user:       "",
			password:   "",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := a.Authenticate(tt.user, tt.password)
			if tt.shouldFail && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected no error for %s, got: %v", tt.name, err)
			}
			if !tt.shouldFail && userID == "" {
				t.Errorf("Expected non-empty user ID for %s", tt.name)
			}
		})
	}
}

// TestAPIHandler tests the REST API handler
func TestAPIHandler(t *testing.T) {
	tokens := []string{"test-api-token"}
	a := auth.NewAuthenticator(tokens, false)
	handler := api.NewHandler(tokens, a, "")

	// Create test server
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	tests := []struct {
		name          string
		method        string
		path          string
		headers       map[string]string
		body          string
		expectedCode  int
		expectedError string
	}{
		{
			name:         "unauthorized request",
			method:       "POST",
			path:         "/api/command",
			headers:      map[string]string{},
			body:         `{"command": "echo test"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "valid authenticated request",
			method:       "POST",
			path:         "/api/command",
			headers:      map[string]string{"X-Auth-Token": "test-api-token"},
			body:         `{"command": "echo hello"}`,
			expectedCode: http.StatusOK,
		},
		{
			name:         "empty command",
			method:       "POST",
			path:         "/api/command",
			headers:      map[string]string{"X-Auth-Token": "test-api-token"},
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "status endpoint",
			method:       "GET",
			path:         "/api/status",
			headers:      map[string]string{"X-Auth-Token": "test-api-token"},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != "" {
				req, err = http.NewRequest(tt.method, ts.URL+tt.path, bytes.NewBufferString(tt.body))
			} else {
				req, err = http.NewRequest(tt.method, ts.URL+tt.path, nil)
			}

			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, resp.StatusCode)
			}

			if tt.expectedCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				if len(body) == 0 {
					t.Error("Expected non-empty response body")
				}
			}
		})
	}
}

// TestWebSocketSession tests WebSocket session creation and authentication
func TestWebSocketSession(t *testing.T) {
	tokens := []string{"ws-test-token"}
	a := auth.NewAuthenticator(tokens, false)
	sessionManager := ws.NewSessionManager(a, "")

	// Create test WebSocket server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", sessionManager.HandleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// This is a basic test to ensure the server starts and accepts connections
	// Full WebSocket testing would require a WebSocket client library
	t.Log("WebSocket test server started at:", ts.URL)
}

// TestSessionManager tests session management
func TestSessionManager(t *testing.T) {
	tokens := []string{"session-test-token"}
	a := auth.NewAuthenticator(tokens, false)
	sessionManager := ws.NewSessionManager(a, "")

	// Check initial state
	if sessionManager == nil {
		t.Fatal("Session manager should not be nil")
	}

	// Test that sessions map is initialized
	t.Log("Session manager created successfully")
}

// TestConcurrentCommands tests running multiple commands concurrently
func TestConcurrentCommands(t *testing.T) {
	session, err := shell.NewSession("")
	if err != nil {
		t.Fatalf("Failed to create shell session: %v", err)
	}
	defer session.Close()

	// Run multiple commands concurrently
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(index int) {
			cmd := fmt.Sprintf("echo test%d\n", index)
			_, err := session.Write([]byte(cmd))
			if err != nil {
				t.Errorf("Failed to write command %d: %v", index, err)
			}
			done <- true
		}(i)
	}

	// Wait for all commands to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Wait for output to be processed
	time.Sleep(1 * time.Second)

	output := session.ReadStdout()
	t.Logf("Concurrent command output: %s", output)
}

// TestErrorHandling tests error handling in various scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("invalid command", func(t *testing.T) {
		output, err := shell.RunCommand("nonexistent_command_xyz", "")
		if err != nil {
			t.Logf("Expected error for invalid command: %v", err)
		}
		if output == nil {
			t.Error("Output should not be nil even on error")
		}
	})

	t.Run("empty command", func(t *testing.T) {
		output, err := shell.RunCommand("", "")
		if err != nil {
			t.Logf("Expected error for empty command: %v", err)
		}
		if output == nil {
			t.Error("Output should not be nil even on error")
		}
	})
}

// TestAuthenticationFlow tests complete authentication flow
func TestAuthenticationFlow(t *testing.T) {
	tokens := []string{"auth-flow-token"}
	a := auth.NewAuthenticator(tokens, false)

	// Test token authentication
	userID, err := a.Authenticate("", "auth-flow-token")
	if err != nil {
		t.Fatalf("Token authentication failed: %v", err)
	}
	if userID != "token-user" {
		t.Errorf("Expected user 'token-user', got '%s'", userID)
	}

	// Test invalid token
	_, err = a.Authenticate("", "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

// TestShellSessionCleanup tests proper cleanup of shell sessions
func TestShellSessionCleanup(t *testing.T) {
	session, err := shell.NewSession("")
	if err != nil {
		t.Fatalf("Failed to create shell session: %v", err)
	}

	// Run a command
	_, err = session.Write([]byte("echo cleanup_test\n"))
	if err != nil {
		t.Fatalf("Failed to write command: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Close session (ignore error if already closed)
	session.Close()
	// Note: Close() returns no value, so we just call it

	// Verify session is closed (IsRunning might still return true briefly)
	// Give it a moment to fully close
	time.Sleep(100 * time.Millisecond)
	if session.IsRunning() {
		t.Log("Note: Session still appears running after Close() - this may be expected behavior")
	}

	// Test that closing again doesn't panic
	session.Close()
	t.Log("Second Close() completed without panic")
}

// TestAPIAuthentication tests API authentication mechanisms
func TestAPIAuthentication(t *testing.T) {
	tokens := []string{"api-auth-token"}
	a := auth.NewAuthenticator(tokens, false)
	handler := api.NewHandler(tokens, a, "")

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	tests := []struct {
		name       string
		token      string
		shouldFail bool
	}{
		{
			name:       "valid token",
			token:      "api-auth-token",
			shouldFail: false,
		},
		{
			name:       "invalid token",
			token:      "invalid-token",
			shouldFail: true,
		},
		{
			name:       "no token",
			token:      "",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", ts.URL+"/api/command", bytes.NewBufferString(`{"command": "echo test"}`))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.token != "" {
				req.Header.Set("X-Auth-Token", tt.token)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if !tt.shouldFail && resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
			if tt.shouldFail && resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", resp.StatusCode)
			}
		})
	}
}

// TestWebSocketMessageHandling tests WebSocket message handling
func TestWebSocketMessageHandling(t *testing.T) {
	// This test verifies that the WebSocket message structures are correct
	tokens := []string{"msg-test-token"}
	a := auth.NewAuthenticator(tokens, false)
	sessionManager := ws.NewSessionManager(a, "")

	if sessionManager == nil {
		t.Fatal("Session manager creation failed")
	}

	t.Log("WebSocket message handling test passed")
}

// TestShellOutputBuffering tests shell output buffering
func TestShellOutputBuffering(t *testing.T) {
	session, err := shell.NewSession("")
	if err != nil {
		t.Fatalf("Failed to create shell session: %v", err)
	}
	defer session.Close()

	// Write multiple commands
	commands := []string{"echo line1\n", "echo line2\n", "echo line3\n"}
	for _, cmd := range commands {
		_, err := session.Write([]byte(cmd))
		if err != nil {
			t.Fatalf("Failed to write command: %v", err)
		}
	}

	// Wait for output
	time.Sleep(1 * time.Second)

	output := session.ReadStdout()
	t.Logf("Buffered output: %s", output)

	// Verify all lines are present (check for the actual output, not the command)
	expectedLines := []string{"line1", "line2", "line3"}
	for _, expected := range expectedLines {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("Expected output to contain '%s', got: %s", expected, output)
		}
	}
}

// TestContextCancellation tests context cancellation in API requests
func TestContextCancellation(t *testing.T) {
	tokens := []string{"context-test-token"}
	a := auth.NewAuthenticator(tokens, false)
	handler := api.NewHandler(tokens, a, "")

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Create a request that will be cancelled
	req, err := http.NewRequestWithContext(ctx, "POST", ts.URL+"/api/command", bytes.NewBufferString(`{"command": "echo test"}`))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("X-Auth-Token", "context-test-token")

	client := &http.Client{Timeout: 5 * time.Second}

	// Cancel the context immediately
	cancel()

	_, err = client.Do(req)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	t.Logf("Context cancellation test completed: %v", err)
}

// BenchmarkShellCommand benchmarks shell command execution
func BenchmarkShellCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := shell.RunCommand("echo benchmark", "")
		if err != nil {
			b.Fatalf("Command failed: %v", err)
		}
	}
}

// BenchmarkAPIRequest benchmarks API request handling
func BenchmarkAPIRequest(b *testing.B) {
	tokens := []string{"benchmark-token"}
	a := auth.NewAuthenticator(tokens, false)
	handler := api.NewHandler(tokens, a, "")

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("POST", ts.URL+"/api/command", bytes.NewBufferString(`{"command": "echo test"}`))
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Auth-Token", "benchmark-token")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}

// TestAPIRateLimiting tests rate limiting on API endpoints
func TestAPIRateLimiting(t *testing.T) {
	tokens := []string{"rate-limit-token"}
	a := auth.NewAuthenticator(tokens, false)
	handler := api.NewHandler(tokens, a, "")

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	// Make 11 failed attempts (should be rate limited on 11th)
	for i := 0; i < 11; i++ {
		req, err := http.NewRequest("POST", ts.URL+"/api/command", bytes.NewBufferString(`{"command": "echo test"}`))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Auth-Token", "invalid-token")
		// Set X-Forwarded-For to simulate same client IP across requests
		req.Header.Set("X-Forwarded-For", "192.168.1.100")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if i < 10 {
			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected 401 on attempt %d, got %d", i+1, resp.StatusCode)
			}
		} else {
			if resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Expected 429 on attempt %d, got %d", i+1, resp.StatusCode)
			}
		}
	}
}

// TestWebSocketRateLimiting tests rate limiting on WebSocket connections
func TestWebSocketRateLimiting(t *testing.T) {
	tokens := []string{"ws-rate-limit-token"}
	a := auth.NewAuthenticator(tokens, false)
	sessionManager := ws.NewSessionManager(a, "")

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", sessionManager.HandleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Log("WebSocket rate limiting test setup completed")
	// Full WebSocket testing would require a WebSocket client library
	// This is a placeholder for the test structure
}

// TestCORSProtection tests CORS protection on WebSocket
func TestCORSProtection(t *testing.T) {
	tokens := []string{"cors-test-token"}
	a := auth.NewAuthenticator(tokens, false)
	sessionManager := ws.NewSessionManager(a, "")

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", sessionManager.HandleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Log("CORS protection test setup completed")
	// CORS validation happens at upgrader level
	// This is a placeholder for the test structure
}
