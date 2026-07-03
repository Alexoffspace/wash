package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"WASH/pkg/api"
	"WASH/pkg/auth"
	"WASH/pkg/config"
	"WASH/pkg/ws"
)

//go:embed templates/*
//go:embed static/*
//go:embed static/fonts/*
var embedFS embed.FS

var (
	// Auth token for API access (comma-separated for multiple)
	authToken = flag.String("token", "", "Auth token for API access (comma-separated for multiple)")

	// Server port
	port = flag.Int("port", 8080, "Server port")

	// Enable OS user/password authentication
	enableOSAuth = flag.Bool("os-auth", false, "Enable OS user/password authentication")

	// Allow listening on 0.0.0.0
	allow0 = flag.Bool("allow-0", false, "Allow listening on 0.0.0.0 (default: 127.0.0.1)")

	// Maximum WebSocket message size
	maxMessageSize = flag.Int64("max-msg-size", 1<<20, "Max WebSocket message size in bytes (default 1MB)")
)

func main() {
	flag.Parse()

	// Загружаем .env файл если существует
	if err := config.LoadEnvFile(); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}

	// Загружаем конфигурацию из config.yaml если существует
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Warning: failed to load config.yaml: %v", err)
	}

	// Применяем значения из config.yaml с приоритетом CLI аргументов
	// CLI аргументы имеют приоритет, если не указаны в config.yaml
	var tokens []string
	if cfg != nil {
		// Порт: если не указан в CLI, берем из config
		if cfg.Port != nil && *port == 8080 { // 8080 - значение по умолчанию
			*port = *cfg.Port
		}

		// OS Auth: если не указан в CLI, берем из config
		if cfg.OSAuth != nil && !*enableOSAuth {
			*enableOSAuth = *cfg.OSAuth
		}

		// Allow0: если не указан в CLI, берем из config
		if cfg.Allow0 != nil && !*allow0 {
			*allow0 = *cfg.Allow0
		}

		// Токен: если не указан в CLI, проверяем переменную окружения из config
		if cfg.TokenEnv != "" && *authToken == "" {
			if token := os.Getenv(cfg.TokenEnv); token != "" {
				*authToken = token
			}
		}

		tokens = parseTokens(*authToken)
	} else {
		tokens = parseTokens(*authToken)
	}
	a := auth.NewAuthenticator(tokens, *enableOSAuth)

	// Определяем рабочую директорию и shell для shell-сессий
	workDir := resolveWorkDir(cfg)
	shellCommand := resolveShellCommand(cfg)
	log.Printf("Work directory: %s", workDir)
	log.Printf("Shell command: %s", shellCommand)

	sessionManager := ws.NewSessionManagerWithAllow0(a, *allow0, workDir, shellCommand)
	apiHandler := api.NewHandler(tokens, a, workDir)

	// Определяем адрес для прослушивания: allow_0=true => все интерфейсы,
	// allow_0=false => только localhost.
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	if *allow0 {
		addr = fmt.Sprintf("0.0.0.0:%d", *port)
	}

	log.Printf("WASH v0.1.0 starting on %s", addr)
	log.Printf("Auth tokens loaded: %d", len(tokens))
	log.Printf("OS auth enabled: %v", *enableOSAuth)
	if cfg != nil {
		if cfg.TokenEnv != "" {
			log.Printf("Token env var: %s", cfg.TokenEnv)
		}
		if cfg.OSAuth != nil {
			log.Printf("OS auth setting: %v", *cfg.OSAuth)
		}
		if cfg.Port != nil && *cfg.Port != 8080 {
			log.Printf("Port setting: %d", *cfg.Port)
		}
		if cfg.Allow0 != nil {
			log.Printf("Allow 0.0.0.0 from config: %v", *cfg.Allow0)
		}
	}
	if *allow0 {
		log.Printf("Allow 0.0.0.0 from CLI: true")
	}

	mux := http.NewServeMux()

	// Web interface
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		file, err := embedFS.ReadFile("templates/index.html")
		if err != nil {
			http.Error(w, "Template not found", http.StatusInternalServerError)
			return
		}

		lightBgs, _ := embedFS.ReadDir("static/backgrounds/light")
		darkBgs, _ := embedFS.ReadDir("static/backgrounds/dark")

		var lightFiles, darkFiles []string
		for _, f := range lightBgs {
			if !f.IsDir() && isImageFile(f.Name()) {
				lightFiles = append(lightFiles, f.Name())
			}
		}
		for _, f := range darkBgs {
			if !f.IsDir() && isImageFile(f.Name()) {
				darkFiles = append(darkFiles, f.Name())
			}
		}

		lightJSON, _ := json.Marshal(lightFiles)
		darkJSON, _ := json.Marshal(darkFiles)

		script := fmt.Sprintf(
			`<script>window.__lightBgs=%s;window.__darkBgs=%s;</script>`,
			lightJSON, darkJSON,
		)
		html := strings.ReplaceAll(string(file), "</head>", script+"</head>")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		sessionManager.HandleWebSocket(w, r)
	})

	// REST API
	apiHandler.RegisterRoutes(mux)

	// Register MIME types for embedded static files
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".jpg", "image/jpeg")
	mime.AddExtensionType(".jpeg", "image/jpeg")
	mime.AddExtensionType(".png", "image/png")
	mime.AddExtensionType(".webp", "image/webp")

	// Static files (CSS, JS) — embedded
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.FS(embedFS)).ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		server.Close()
	}()

	log.Fatal(server.ListenAndServe())
}

func parseTokens(tokenStr string) []string {
	if tokenStr == "" {
		return nil
	}
	var tokens []string
	for _, t := range splitCSV(tokenStr) {
		t = trimSpace(t)
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func splitCSV(s string) []string {
	var result []string
	var current string
	inQuote := false
	for _, c := range s {
		switch c {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				result = append(result, current)
				current = ""
				continue
			}
			fallthrough
		default:
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// resolveWorkDir возвращает рабочую директорию из конфига или home пользователя
func resolveWorkDir(cfg *config.Config) string {
	if cfg != nil && cfg.WorkDir != "" {
		return cfg.WorkDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: cannot determine home directory: %v", err)
		return ""
	}
	return homeDir
}

// resolveShellCommand возвращает команду shell из конфига или платформенный дефолт
func resolveShellCommand(cfg *config.Config) string {
	if cfg != nil && cfg.ShellCommand != "" {
		return cfg.ShellCommand
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "sh"
}

func isImageFile(name string) bool {
	idx := strings.LastIndex(name, ".")
	if idx == -1 {
		return false
	}
	ext := strings.ToLower(name[idx+1:])
	switch ext {
	case "jpg", "jpeg", "png", "webp":
		return true
	}
	return false
}
