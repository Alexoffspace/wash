package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию приложения
// Config представляет конфигурацию приложения
type Config struct {
	// OSAuth - включить OS аутентификацию
	OSAuth *bool `yaml:"os_auth"`

	// TokenEnv - имя переменной окружения для токена
	TokenEnv string `yaml:"token"`

	// Port - порт сервера
	Port *int `yaml:"port"`

	// Allow0 - слушать на 0.0.0.0 (true) или 127.0.0.1 (false)
	Allow0 *bool `yaml:"allow_0"`

	// WorkDir - рабочая директория для shell-сессий и команд
	// Если не указана, используется home-директория пользователя
	WorkDir string `yaml:"work_dir"`
}

// LoadConfig пытается загрузить конфигурацию из config.yaml
// Возвращает nil, если файл не найден (это не ошибка)
func LoadConfig() (*Config, error) {
	configPath := "config.yaml"

	// Проверяем существование файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // Файл не найден - это нормально
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	return &cfg, nil
}

// LoadEnvFile загружает переменные окружения из .env файла
func LoadEnvFile() error {
	envPath := ".env"

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil // .env файл не обязателен
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("failed to read .env: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Ищем разделитель =
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Удаляем кавычки если есть
		value = strings.Trim(value, "\"'")

		// Устанавливаем переменную окружения
		os.Setenv(key, value)
	}

	return nil
}

// GetEnvValue возвращает значение переменной окружения
// Если переменная не найдена, возвращает пустую строку
func GetEnvValue(name string) string {
	return os.Getenv(name)
}

// GetRequiredEnvValue возвращает значение переменной окружения
// Если переменная не найдена, возвращает ошибку
func GetRequiredEnvValue(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", name)
	}
	return value, nil
}

// ResolveEnvVars подставляет значения переменных окружения из списка
// Возвращает карту имя -> значение для существующих переменных
func ResolveEnvVars(envVars []string) map[string]string {
	result := make(map[string]string)
	for _, name := range envVars {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if value := os.Getenv(name); value != "" {
			result[name] = value
		}
	}
	return result
}

// GetExecutablePath возвращает путь к исполняемому файлу приложения
func GetExecutablePath() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(ex)
}

// GetProjectRoot ищет корень проекта, поднимаясь вверх по директориям
// Ищет файлы: go.mod, package.json, .git
func GetProjectRoot() string {
	current := GetExecutablePath()
	maxDepth := 10 // Ограничение глубины поиска

	for i := 0; i < maxDepth; i++ {
		// Проверяем наличие маркеров корня проекта
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			break // Достигли корня файловой системы
		}
		current = parent
	}

	return ""
}
