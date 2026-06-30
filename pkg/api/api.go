package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"WASH/pkg/auth"
	"WASH/pkg/shell"
)

// Handler handles REST API requests
type Handler struct {
	AuthHandler
	authAttempts map[string]*APIAuthAttemptTracker
	authMu       sync.RWMutex
	WorkDir      string
}

// AuthHandler is responsible for authentication
type AuthHandler struct {
	tokens        []string
	authenticator *auth.Authenticator
}

// APIAuthAttemptTracker tracks authentication attempts for rate limiting
type APIAuthAttemptTracker struct {
	attempts  int
	lastReset time.Time
}

// NewHandler creates a REST API handler
func NewHandler(tokens []string, authenticator *auth.Authenticator, workDir string) *Handler {
	return &Handler{
		AuthHandler: AuthHandler{
			tokens:        tokens,
			authenticator: authenticator,
		},
		authAttempts: make(map[string]*APIAuthAttemptTracker),
		WorkDir:      workDir,
	}
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// CommandRequest represents a command execution request
type CommandRequest struct {
	Command string `json:"command"`
}

// CommandResponse represents a response with command result
type CommandResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// Authenticate checks credentials
func (h *AuthHandler) Authenticate(r *http.Request) (string, error) {
	// Check token in header
	token := r.Header.Get("X-Auth-Token")
	if token != "" {
		for _, t := range h.tokens {
			if token == t {
				return "token-user", nil
			}
		}
		// Token not matched — fall through to Basic Auth (supports OS auth)
	}

	// Check Basic Auth
	user, pass, ok := r.BasicAuth()
	if ok && user != "" && pass != "" {
		// Use Authenticator for full credential verification (tokens + OS auth)
		if h.authenticator != nil {
			return h.authenticator.Authenticate(user, pass)
		}

		// Fallback if authenticator not available (token-only mode)
		for _, t := range h.tokens {
			if pass == t {
				return "token-user", nil
			}
		}
		return user, nil
	}

	return "", fmt.Errorf("missing or invalid authentication")
}

// HandleCommand handles POST /api/command
func (h *Handler) HandleCommand(w http.ResponseWriter, r *http.Request) {
	// Get client IP for rate limiting
	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = strings.Split(xff, ",")[0]
	}

	// Check rate limiting before authentication
	if h.isAuthAttemptLimited(clientIP) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Too many authentication attempts. Please try again later."})
		return
	}

	// Authentication
	user, err := h.Authenticate(r)
	if err != nil {
		h.recordAuthAttempt(clientIP, false)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Record successful auth attempt
	h.recordAuthAttempt(clientIP, true)

	// Parse request
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Command == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "command is required"})
		return
	}

	// Execute command
	output, err := shell.RunCommand(req.Command, h.WorkDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CommandResponse{
			Error: err.Error(),
		})
		return
	}

	log.Printf("Command executed by '%s': %s", user, req.Command)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CommandResponse{
		Stdout:   output.Stdout,
		Stderr:   output.Stderr,
		ExitCode: output.ExitCode,
	})
}

// SystemInfoResponse represents system information response
type SystemInfoResponse struct {
	Hostname     string         `json:"hostname"`
	IPAddresses  []string       `json:"ip_addresses"`
	OS           string         `json:"os"`
	OSVersion    string         `json:"os_version"`
	Architecture string         `json:"architecture"`
	Time         string         `json:"time"`
	Uptime       string         `json:"uptime"`
	WASHStatus   WASHStatusInfo `json:"wash_status"`
	CPU          CPUInfo        `json:"cpu"`
	Memory       MemoryInfo     `json:"memory"`
	Disk         DiskInfo       `json:"disk"`
	Processes    int            `json:"processes"`
	User         string         `json:"user"`
}

// WASHStatusInfo represents WASH application status
type WASHStatusInfo struct {
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	Version    string    `json:"version"`
	AuthTokens int       `json:"auth_tokens"`
}

// CPUInfo represents CPU information
type CPUInfo struct {
	Load1    string  `json:"load1"`     // 1-minute load average
	Load5    string  `json:"load5"`     // 5-minute load average
	Load15   string  `json:"load15"`    // 15-minute load average
	Cores    int     `json:"cores"`     // Number of CPU cores
	UsagePct float64 `json:"usage_pct"` // Real-time CPU usage % (-1 if unavailable)
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total     uint64  `json:"total"`      // Total RAM in bytes
	Used      uint64  `json:"used"`       // Used RAM in bytes
	Free      uint64  `json:"free"`       // Free RAM in bytes
	UsedPct   float64 `json:"used_pct"`   // Percentage used
	SwapTotal uint64  `json:"swap_total"` // Total swap in bytes
	SwapUsed  uint64  `json:"swap_used"`  // Used swap in bytes
}

// DiskInfo represents disk information
type DiskInfo struct {
	Total      uint64  `json:"total"`       // Total disk space in bytes
	Used       uint64  `json:"used"`        // Used disk space in bytes
	Free       uint64  `json:"free"`        // Free disk space in bytes
	UsedPct    float64 `json:"used_pct"`    // Percentage used
	MountPoint string  `json:"mount_point"` // Mount point (usually /)
}

// HandleStatus handles GET /api/status - system info endpoint
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	// Get client IP for rate limiting
	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = strings.Split(xff, ",")[0]
	}

	// Check rate limiting before authentication
	if h.isAuthAttemptLimited(clientIP) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Too many authentication attempts. Please try again later."})
		return
	}

	// Authentication
	_, err := h.Authenticate(r)
	if err != nil {
		h.recordAuthAttempt(clientIP, false)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Record successful auth attempt
	h.recordAuthAttempt(clientIP, true)

	w.Header().Set("Content-Type", "application/json")

	info := SystemInfoResponse{
		WASHStatus: WASHStatusInfo{
			Status:     "running",
			StartTime:  time.Now().Add(-1 * time.Hour), // TODO: track actual start time
			Version:    "0.1.0",
			AuthTokens: len(h.tokens),
		},
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	info.Hostname = hostname

	// Get IP addresses
	info.IPAddresses = getIPAddresses()

	// Get OS info
	info.OS = runtime.GOOS
	info.Architecture = runtime.GOARCH
	info.OSVersion = getOSVersion()

	// Get current time
	info.Time = time.Now().Format("2006-01-02 15:04:05 MST")

	// Get uptime
	info.Uptime = getSystemUptime()

	// Get CPU info
	info.CPU = getCPUInfo()
	info.CPU.UsagePct = getRealCPUUsage()

	// Get memory info
	info.Memory = getMemoryInfo()

	// Get disk info
	info.Disk = getDiskInfo()

	// Get process count
	info.Processes = getProcessCount()

	// Get current user
	info.User = os.Getenv("USER")
	if info.User == "" {
		info.User = os.Getenv("USERNAME")
	}
	if info.User == "" {
		info.User = "unknown"
	}

	json.NewEncoder(w).Encode(info)
}

// getIPAddresses returns all IP addresses of the system with subnet mask
func getIPAddresses() []string {
	var addresses []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{"error"}
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				addresses = append(addresses, ipNet.String())
			}
		}
	}
	if len(addresses) == 0 {
		addresses = []string{"127.0.0.1/8"}
	}
	return addresses
}

// getOSVersion returns OS version string
func getOSVersion() string {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "os", "get", "Caption", "/value")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Caption=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
		return "unknown"
	}
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// getSystemUptime returns system uptime string
func getSystemUptime() string {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-Command", "(Get-Date).ToString('yyyy-MM-dd HH:mm:ss')")
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
		return "unknown"
	}
	cmd := exec.Command("uptime")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// getCPUInfo returns CPU information
func getCPUInfo() CPUInfo {
	info := CPUInfo{
		Cores: runtime.NumCPU(),
	}

	if runtime.GOOS == "windows" {
		info.Load1 = "N/A"
		info.Load5 = "N/A"
		info.Load15 = "N/A"
		return info
	}

	// Try to get load averages
	cmd := exec.Command("uptime")
	output, err := cmd.Output()
	if err == nil {
		outputStr := string(output)
		// Parse load averages from uptime output
		if strings.Contains(outputStr, "load average") {
			parts := strings.Split(outputStr, "load average:")
			if len(parts) > 1 {
				loadStr := strings.TrimSpace(parts[1])
				// Handle locale where comma is both decimal and list separator (e.g. "2,07, 1,63, 1,54")
				loadStr = strings.ReplaceAll(loadStr, ", ", " ")
				loadStr = strings.ReplaceAll(loadStr, ",", ".")
				loadParts := strings.Fields(loadStr)
				if len(loadParts) >= 3 {
					info.Load1 = loadParts[0]
					info.Load5 = loadParts[1]
					info.Load15 = loadParts[2]
				}
			}
		}
	}

	return info
}

// Package-level cache for real CPU usage calculation via /proc/stat
var (
	prevCPUTotal uint64
	prevCPUIdle  uint64
	cpuMu        sync.Mutex
	cpuFirstCall bool = true
)

// getRealCPUUsage reads /proc/stat and calculates actual CPU usage percentage.
// Returns -1 if unavailable (non-Linux or first call).
func getRealCPUUsage() float64 {
	if runtime.GOOS != "linux" {
		return -1
	}

	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return -1
	}

	var total, idle uint64
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return -1
		}
		for i := 1; i < len(fields); i++ {
			val, _ := strconv.ParseUint(fields[i], 10, 64)
			total += val
			if i == 4 {
				idle = val
			}
		}
		break
	}

	cpuMu.Lock()
	defer cpuMu.Unlock()

	if cpuFirstCall {
		prevCPUTotal = total
		prevCPUIdle = idle
		cpuFirstCall = false
		return -1
	}

	deltaTotal := total - prevCPUTotal
	deltaIdle := idle - prevCPUIdle

	prevCPUTotal = total
	prevCPUIdle = idle

	if deltaTotal == 0 {
		return 0
	}

	return float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
}

// getMemoryInfo returns memory information
func getMemoryInfo() MemoryInfo {
	info := MemoryInfo{}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	info.Total = memStats.TotalAlloc
	info.Used = memStats.Alloc
	info.Free = memStats.TotalAlloc - memStats.Alloc

	if info.Total > 0 {
		info.UsedPct = float64(info.Used) / float64(info.Total) * 100
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-Command", "[System.Runtime.InteropServices.Marshal]::PtrToStringAuto([System.Runtime.InteropServices.Marshal]::StringToBSTR([System.Diagnostics.Process]::GetCurrentProcess().WorkingSet))")
		output, err := cmd.Output()
		if err == nil {
			if size, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64); err == nil {
				info.Used = size
			}
		}
		return info
	}

	// Try to get detailed memory info from system
	cmd := exec.Command("free", "-b")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			parts := strings.Fields(lines[1])
			if len(parts) >= 7 {
				if total, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					info.Total = total
				}
				if used, err := strconv.ParseUint(parts[2], 10, 64); err == nil {
					info.Used = used
				}
				if free, err := strconv.ParseUint(parts[3], 10, 64); err == nil {
					info.Free = free
				}
				if swapTotal, err := strconv.ParseUint(parts[6], 10, 64); err == nil {
					info.SwapTotal = swapTotal
				}
			}
		}

		// Recalculate percentage
		if info.Total > 0 {
			info.UsedPct = float64(info.Used) / float64(info.Total) * 100
		}
	}

	return info
}

// getDiskInfo returns disk information
func getDiskInfo() DiskInfo {
	info := DiskInfo{
		MountPoint: "/",
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-Command", "Get-Volume | Select-Object DriveLetter, Size, SizeRemaining | ConvertTo-Json")
		output, err := cmd.Output()
		if err == nil {
			// Parse JSON output
			var volumes []map[string]interface{}
			if err := json.Unmarshal(output, &volumes); err == nil && len(volumes) > 0 {
				vol := volumes[0]
				if size, ok := vol["Size"].(float64); ok {
					info.Total = uint64(size)
				}
				if remaining, ok := vol["SizeRemaining"].(float64); ok {
					info.Free = uint64(remaining)
					info.Used = info.Total - info.Free
					if info.Total > 0 {
						info.UsedPct = float64(info.Used) / float64(info.Total) * 100
					}
				}
				if driveLetter, ok := vol["DriveLetter"].(string); ok {
					info.MountPoint = driveLetter + ":"
				}
			}
		}
		return info
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		info.Total = stat.Blocks * uint64(stat.Bsize)
		info.Free = stat.Bfree * uint64(stat.Bsize)
		info.Used = info.Total - info.Free

		if info.Total > 0 {
			info.UsedPct = float64(info.Used) / float64(info.Total) * 100
		}
	}

	return info
}

// getProcessCount returns approximate process count
func getProcessCount() int {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-Command", "(Get-Process).Count")
		output, err := cmd.Output()
		if err == nil {
			count, err := strconv.Atoi(strings.TrimSpace(string(output)))
			if err == nil {
				return count
			}
		}
		return 0
	}
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) > 1 {
		return len(lines) - 1
	}
	return 0
}

// isAuthAttemptLimited checks if client has exceeded rate limit
func (h *Handler) isAuthAttemptLimited(clientIP string) bool {
	h.authMu.RLock()
	tracker, exists := h.authAttempts[clientIP]
	h.authMu.RUnlock()

	if !exists {
		return false
	}

	// Reset counter if more than 1 minute has passed
	if time.Since(tracker.lastReset) > 1*time.Minute {
		h.authMu.Lock()
		delete(h.authAttempts, clientIP)
		h.authMu.Unlock()
		return false
	}

	// Allow max 10 attempts per minute
	return tracker.attempts >= 10
}

// recordAuthAttempt records an authentication attempt for rate limiting
func (h *Handler) recordAuthAttempt(clientIP string, success bool) {
	h.authMu.Lock()
	defer h.authMu.Unlock()

	tracker, exists := h.authAttempts[clientIP]
	if !exists {
		tracker = &APIAuthAttemptTracker{
			lastReset: time.Now(),
		}
		h.authAttempts[clientIP] = tracker
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
		delete(h.authAttempts, clientIP)
	}

	if tracker.attempts > 0 {
		log.Printf("API auth attempt %d for %s (success: %v)", tracker.attempts, clientIP, success)
	}
}

// RegisterRoutes registers API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/command", h.HandleCommand)
	mux.HandleFunc("/api/status", h.HandleStatus)
}
