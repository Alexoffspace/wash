package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"WASH/pkg/auth"
	"WASH/pkg/shell"
)

func newUpgrader(allow0 bool) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			allowed := isAllowedOrigin(r, allow0)
			if !allowed {
				log.Printf("WebSocket connection rejected: disallowed origin %s", r.Header.Get("Origin"))
			}
			return allowed
		},
	}
}

func isAllowedOrigin(r *http.Request, allow0 bool) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients often omit Origin. Authentication is still required.
		return true
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	originHost := stripPort(u.Host)
	requestHost := stripPort(r.Host)

	if allow0 {
		// When listening on all interfaces, allow browser WebSocket connections only
		// from the same host that received the HTTP/WebSocket request. This permits
		// fishnet.vita.local, LAN IPs, etc. without allowing arbitrary cross-site origins.
		return strings.EqualFold(u.Host, r.Host)
	}

	// In localhost-only mode, accept only loopback origins and requests.
	return isLoopbackHost(originHost) && isLoopbackHost(requestHost)
}

func stripPort(hostport string) string {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	return strings.Trim(host, "[]")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// SessionManager manages WebSocket sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	auth     *auth.Authenticator

	upgrader websocket.Upgrader

	// Rate limiting for authentication attempts
	authAttempts map[string]*AuthAttemptTracker
	authMu       sync.RWMutex

	// WorkDir — рабочая директория для shell-сессий
	WorkDir string

	// ShellCommand — команда для запуска shell (sh, bash, zsh...)
	ShellCommand string
}

// AuthAttemptTracker tracks authentication attempts for rate limiting
type AuthAttemptTracker struct {
	attempts  int
	lastReset time.Time
}

// Session represents a WebSocket session
type Session struct {
	ID        string
	UserID    string
	Conn      *websocket.Conn
	Send      chan []byte
	Shell     *shell.Session
	mu        sync.Mutex
	closed    bool
	createdAt time.Time
}

// NewSession creates a new session
func NewSession(id, userID string, conn *websocket.Conn) *Session {
	log.Printf("[SESSION %s] NewSession: creating session with buffered channel", id)
	return &Session{
		ID:        id,
		UserID:    userID,
		Conn:      conn,
		Send:      make(chan []byte, 100), // Large buffer to prevent blocking
		createdAt: time.Now(),
	}
}

// NewSessionManager creates a session manager in localhost-only mode.
func NewSessionManager(auth *auth.Authenticator, workDir string, shellCommand string) *SessionManager {
	return NewSessionManagerWithAllow0(auth, false, workDir, shellCommand)
}

// NewSessionManagerWithAllow0 creates a session manager with WebSocket origin
// checks aligned to the server bind mode.
func NewSessionManagerWithAllow0(auth *auth.Authenticator, allow0 bool, workDir string, shellCommand string) *SessionManager {
	return &SessionManager{
		sessions:     make(map[string]*Session),
		auth:         auth,
		upgrader:     newUpgrader(allow0),
		authAttempts: make(map[string]*AuthAttemptTracker),
		WorkDir:      workDir,
		ShellCommand: shellCommand,
	}
}

// HandleWebSocket handles WebSocket connection
func (m *SessionManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get client IP for rate limiting
	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = strings.Split(xff, ",")[0]
	}

	// Set read deadline for first message (5 seconds)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send authentication request
	authReq := &AuthRequest{}
	if err := conn.ReadJSON(authReq); err != nil {
		log.Printf("WebSocket auth timeout/error from %s: %v", clientIP, err)
		m.sendJSON(conn, Message{
			Type:      "auth_error",
			Content:   "Authentication timeout: first message must be sent within 5 seconds",
			Timestamp: time.Now().Format("15:04"),
		})
		conn.Close()
		return
	}

	// Clear read deadline after first message
	conn.SetReadDeadline(time.Time{})

	// Check rate limiting
	if m.isAuthAttemptLimited(clientIP) {
		log.Printf("[SESSION %s] SERVER -> auth_error: rate limited for %s", "initial", clientIP)
		m.sendJSON(conn, Message{
			Type:      "auth_error",
			Content:   "Too many authentication attempts. Please try again later.",
			Timestamp: time.Now().Format("15:04"),
		})
		conn.Close()
		return
	}

	// Authenticate user
	userID, err := m.auth.Authenticate(authReq.Login, authReq.Password)
	if err != nil {
		m.recordAuthAttempt(clientIP, false)
		log.Printf("[SESSION %s] SERVER -> auth_error: %v", "initial", err)
		m.sendJSON(conn, Message{
			Type:      "auth_error",
			Content:   fmt.Sprintf("Authentication failed: %v", err),
			Timestamp: time.Now().Format("15:04"),
		})
		conn.Close()
		return
	}

	// Record successful auth attempt
	m.recordAuthAttempt(clientIP, true)

	// Create session
	sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
	log.Printf("[SESSION %s] SERVER -> auth_success: user=%s session=%s", sessionID, userID, sessionID)

	session := NewSession(sessionID, userID, conn)

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	log.Printf("New session: %s (user: %s)", sessionID, userID)

	// Gather system info
	hostname, _ := os.Hostname()
	ip := getLocalIP()

	// Send connection confirmation
	m.sendJSON(conn, AuthResponse{
		Type:     "auth_success",
		Session:  sessionID,
		User:     userID,
		Hostname: hostname,
		IP:       ip,
	})

	// Send system welcome
	m.sendJSON(conn, Message{
		Type:      "system",
		Content:   fmt.Sprintf("Host: %s  |  IP: %s  |  User: %s", hostname, ip, userID),
		Timestamp: time.Now().Format("15:04"),
	})
	log.Printf("[SESSION %s] SERVER -> system: Host=%s IP=%s User=%s", sessionID, hostname, ip, userID)

	// Create shell session
	shellSession, err := shell.NewSession(m.ShellCommand, m.WorkDir, 24, 80)
	if err != nil {
		log.Printf("[SESSION %s] SERVER -> error: Failed to start shell: %v", sessionID, err)
		m.sendError(conn, fmt.Sprintf("Failed to start shell: %v", err))
		m.removeSession(sessionID)
		conn.Close()
		return
	}
	session.Shell = shellSession

	// Send welcome message
	m.sendJSON(conn, Message{
		Type:      "system",
		Content:   fmt.Sprintf("Shell session started for user: %s (session: %s)", userID, sessionID),
		Timestamp: time.Now().Format("15:04"),
	})
	log.Printf("[SESSION %s] SERVER -> system: Shell session started for user=%s", sessionID, userID)

	// Start handlers
	go m.readMessages(conn, session)
	go m.writeMessages(conn, session)
	go m.monitorShell(session)
}

// readMessages reads messages from client
func (m *SessionManager) readMessages(conn *websocket.Conn, session *Session) {
	defer func() {
		m.closeSession(session.ID)
	}()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Session %s read error: %v", session.ID, err)
			}
			break
		}

		log.Printf("[SESSION %s] CLIENT -> %s: %q", session.ID, msg.Type, msg.Content)

		switch msg.Type {
		case "command":
			if session.Shell != nil {
				cmdStr := msg.Content
				if len(cmdStr) > 0 && cmdStr[len(cmdStr)-1] != '\n' {
					cmdStr += "\n"
				}
				if _, err := session.Shell.Write([]byte(cmdStr)); err != nil {
					m.sendError(conn, fmt.Sprintf("Failed to write to shell: %v", err))
				}
				log.Printf("[SESSION %s] SERVER -> command sent to shell", session.ID)
			}

		case "key":
			if session.Shell != nil {
				if _, err := session.Shell.Write([]byte(msg.Content)); err != nil {
					m.sendError(conn, fmt.Sprintf("Failed to write to shell: %v", err))
				}
			}

		case "resize":
			if session.Shell != nil {
				rows, cols := msg.Rows, msg.Cols
				if rows <= 0 {
					rows = 24
				}
				if cols <= 0 {
					cols = 80
				}
				if err := session.Shell.Resize(rows, cols); err != nil {
					log.Printf("[SESSION %s] resize error: %v", session.ID, err)
				}
				log.Printf("[SESSION %s] terminal resized to %dx%d", session.ID, cols, rows)
			}

		case "ping":
			// Respond to ping
			m.sendJSON(conn, Message{
				Type:      "pong",
				Content:   "pong",
				Timestamp: time.Now().Format(time.RFC3339),
			})
			log.Printf("[SESSION %s] SERVER -> pong", session.ID)

		case "pong":
			// Client pong echo, ignore
			log.Printf("[SESSION %s] CLIENT -> pong (ignored)", session.ID)

		default:
			m.sendError(conn, fmt.Sprintf("Unknown message type: %s", msg.Type))
			log.Printf("[SESSION %s] SERVER -> error: Unknown message type: %s", session.ID, msg.Type)
		}
	}
}

// writeMessages sends messages to client
func (m *SessionManager) writeMessages(conn *websocket.Conn, session *Session) {
	defer func() {
		log.Printf("[SESSION %s] writeMessages: exiting", session.ID)
		conn.Close()
	}()

	log.Printf("[SESSION %s] writeMessages: started", session.ID)

	for {
		log.Printf("[SESSION %s] writeMessages: waiting for data from channel", session.ID)

		// Simple blocking read - no select, no timeout
		data, ok := <-session.Send
		log.Printf("[SESSION %s] writeMessages: RECEIVED from channel, ok=%v, len=%d", session.ID, ok, len(data))

		if !ok {
			log.Printf("[SESSION %s] writeMessages: session.Send channel closed", session.ID)
			return
		}

		session.mu.Lock()
		if session.closed {
			log.Printf("[SESSION %s] writeMessages: session already closed", session.ID)
			session.mu.Unlock()
			return
		}
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		session.mu.Unlock()

		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[SESSION %s] WRITE ERROR: %v", session.ID, err)
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err == nil {
			log.Printf("[SESSION %s] SERVER -> %s: %q", session.ID, msg.Type, msg.Content)
		} else {
			log.Printf("[SESSION %s] SERVER -> raw: %s", session.ID, string(data))
		}
	}
}

// monitorShell reads shell output and sends it to the WebSocket client
func (m *SessionManager) monitorShell(session *Session) {
	log.Printf("[SESSION %s] monitorShell: started (PTY mode)", session.ID)

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case data, ok := <-session.Shell.Output():
			if !ok {
				log.Printf("[SESSION %s] monitorShell: shell output channel closed", session.ID)
				m.closeSession(session.ID)
				return
			}
			msg := Message{
				Type:    "output",
				Content: string(data),
			}
			session.Send <- mustJSON(msg)

		case <-pingTicker.C:
			msg := Message{
				Type:    "ping",
				Content: "keepalive",
			}
			select {
			case session.Send <- mustJSON(msg):
			default:
			}
		}
	}
}

// closeSession closes a session
func (m *SessionManager) closeSession(sessionID string) {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	log.Printf("Closing session: %s (user: %s)", sessionID, session.UserID)

	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return
	}
	session.closed = true
	// Close shell and connection first
	session.mu.Unlock()

	if session.Shell != nil {
		session.Shell.Close()
	}
	session.Conn.Close()

	// Close send channel last, after shell and connection are closed
	close(session.Send)

	m.removeSession(sessionID)
}

// removeSession removes a session from the manager
func (m *SessionManager) removeSession(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

// getLocalIP returns the first non-loopback, non-virtual IPv4 address of the host
func getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
	for _, iface := range ifaces {
		// Skip down, loopback, and virtual interfaces (docker, veth, tun, etc.)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifName := iface.Name
		// Skip common virtual interface names
		if strings.HasPrefix(ifName, "docker") || strings.HasPrefix(ifName, "veth") ||
			strings.HasPrefix(ifName, "tun") || strings.HasPrefix(ifName, "tap") ||
			strings.HasPrefix(ifName, "lo") || strings.HasPrefix(ifName, "br-") ||
			strings.HasPrefix(ifName, "vboxnet") || strings.HasPrefix(ifName, "vmnet") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ip4 := ipNet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return "unknown"
}

// sendJSON sends a JSON message directly to the connection (use with caution - can cause concurrent writes)
func (m *SessionManager) sendJSON(conn *websocket.Conn, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("WebSocket write error: %v", err)
	}
}

// mustJSON marshals a message to JSON, panics on error
func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return data
}

// sendError sends an error message
func (m *SessionManager) sendError(conn *websocket.Conn, errMsg string) {
	m.sendJSON(conn, Message{
		Type:      "error",
		Content:   errMsg,
		Timestamp: time.Now().Format("15:04"),
	})
}

// isAuthAttemptLimited checks if client has exceeded rate limit
func (m *SessionManager) isAuthAttemptLimited(clientIP string) bool {
	m.authMu.RLock()
	tracker, exists := m.authAttempts[clientIP]
	m.authMu.RUnlock()

	if !exists {
		return false
	}

	// Reset counter if more than 1 minute has passed
	if time.Since(tracker.lastReset) > 1*time.Minute {
		m.authMu.Lock()
		delete(m.authAttempts, clientIP)
		m.authMu.Unlock()
		return false
	}

	// Allow max 10 attempts per minute
	return tracker.attempts >= 10
}

// recordAuthAttempt records an authentication attempt for rate limiting
func (m *SessionManager) recordAuthAttempt(clientIP string, success bool) {
	m.authMu.Lock()
	defer m.authMu.Unlock()

	tracker, exists := m.authAttempts[clientIP]
	if !exists {
		tracker = &AuthAttemptTracker{
			lastReset: time.Now(),
		}
		m.authAttempts[clientIP] = tracker
	}

	// Reset counter if more than 1 minute has passed
	if time.Since(tracker.lastReset) > 1*time.Minute {
		tracker.attempts = 0
		tracker.lastReset = time.Now()
	}

	// Only count failed attempts
	if !success {
		tracker.attempts++
	} else {
		// Reset on successful auth
		delete(m.authAttempts, clientIP)
	}

	if tracker.attempts > 0 {
		log.Printf("Auth attempt %d for %s (success: %v)", tracker.attempts, clientIP, success)
	}
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	Type     string `json:"type"`
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Type     string `json:"type"`
	Session  string `json:"session,omitempty"`
	User     string `json:"user,omitempty"`
	Error    string `json:"error,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	IP       string `json:"ip,omitempty"`
}

// Message represents a WebSocket message
type Message struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	Session   string `json:"session,omitempty"`
	User      string `json:"user,omitempty"`
	Timestamp string `json:"timestamp"`
	Error     string `json:"error,omitempty"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
}
