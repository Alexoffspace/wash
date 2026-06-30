# Руководство по автотестам WASH

Это руководство содержит информацию о запуске и использовании автотестов для проекта WASH.

## Расположение файлов

- **Файл тестов**: `integration_test.go`
- **Путь**: Корень проекта (`WAShell/integration_test.go`)

## Предварительные требования

- Go 1.22 или выше
- Проект должен быть успешно собран перед запуском тестов

## Запуск тестов

### 1. Запустить все тесты

```bash
cd WAShell
go test -v ./...
```

### 2. Запустить только интеграционные тесты

```bash
go test -v -run .
```

### 3. Запустить с отчётом о покрытии кода

```bash
go test -v -cover ./...
```

### 4. Запустить конкретный тест

```bash
# Пример: запустить только тест ShellSession
go test -v -run TestShellSession
```

### 5. Запустить бенчмарки (производительность)

```bash
# Запустить все бенчмарки
go test -bench=. -benchmem

# Запустить конкретный бенчмарк
go test -bench=BenchmarkShellCommand -benchmem
```

### 6. Запустить тесты с таймаутом

```bash
# Установить таймаут 30 секунд для всех тестов
go test -timeout 30s ./...
```

## Структура тестов

### Тесты оболочки (Shell Tests)

| Тест | Описание |
|------|----------|
| `TestShellSession` | Базовая работа с сессией оболочки (создание, запись, чтение вывода) |
| `TestRunCommand` | Выполнение команд через `RunCommand` с различными командами |
| `TestConcurrentCommands` | Одновременное выполнение нескольких команд |
| `TestShellOutputBuffering` | Проверка буферизации вывода при множественных командах |
| `TestShellSessionCleanup` | Корректное завершение и очистка сессии оболочки |

### Тесты аутентификации (Auth Tests)

| Тест | Описание |
|------|----------|
| `TestAuthenticator` | Проверка работы аутентификатора с токенами |
| `TestAuthenticationFlow` | Полный поток аутентификации |
| `TestAPIAuthentication` | Проверка механизмов аутентификации API |

### Тесты REST API (API Tests)

| Тест | Описание |
|------|----------|
| `TestAPIHandler` | Тестирование основных эндпоинтов API |
| `TestAPIAuthentication` | Проверка авторизации через токены |

### Тесты WebSocket (WebSocket Tests)

| Тест | Описание |
|------|----------|
| `TestWebSocketSession` | Создание и проверка WebSocket сессии |
| `TestSessionManager` | Управление сессиями |
| `TestWebSocketMessageHandling` | Обработка WebSocket сообщений |

### Тесты обработки ошибок (Error Handling Tests)

| Тест | Описание |
|------|----------|
| `TestErrorHandling` | Обработка некорректных команд и пустых команд |

### Тесты контекста (Context Tests)

| Тест | Описание |
|------|----------|
| `TestContextCancellation` | Отмена контекста в API запросах |

### Бенчмарки (Benchmarks)

| Бенчмарк | Описание |
|----------|----------|
| `BenchmarkShellCommand` | Производительность выполнения команд оболочки |
| `BenchmarkAPIRequest` | Производительность обработки API запросов |

## Примеры использования

### Запуск тестов в CI/CD

```bash
# Установить переменные окружения
export GO111MODULE=on

# Запустить тесты с покрытием
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Запуск тестов с verbose выводом

```bash
go test -v -count=1 ./...
```

### Запуск тестов с параллелизмом

```bash
go test -v -parallel=4 ./...
```

### Запуск тестов с фильтром по имени

```bash
# Запустить все тесты, содержащие "Auth"
go test -v -run "Auth" ./...

# Запустить все тесты, содержащие "Shell"
go test -v -run "Shell" ./...
```

## Интерпретация результатов

### Успешный запуск

```
=== RUN   TestShellSession
--- PASS: TestShellSession (0.50s)
=== RUN   TestRunCommand
=== RUN   TestRunCommand/echo_command
--- PASS: TestRunCommand (0.10s)
    --- PASS: TestRunCommand/echo_command (0.05s)
PASS
ok      WASH    0.600s
```

### Неуспешный запуск

```
=== RUN   TestShellSession
    integration_test.go:15: Failed to create shell session: failed to start shell: exec: "sh": executable file not found in $PATH
--- FAIL: TestShellSession (0.00s)
FAIL
exit status 1
FAIL    WASH    0.002s
```

## Частые проблемы и решения

### Проблема: Тесты не запускаются

**Решение:** Убедитесь, что вы находитесь в корне проекта:
```bash
cd WAShell
go mod tidy
go build
go test -v ./...
```

### Проблема: Тесты падают с ошибкой "executable file not found"

**Решение:** Проверьте, что оболочка установлена:
```bash
which sh
which bash
```

### Проблема: Тесты занимают слишком много времени

**Решение:** Установите таймаут для тестов:
```bash
go test -timeout 30s ./...
```

### Проблема: Непредсказуемые результаты тестов

**Решение:** Запустите тесты несколько раз:
```bash
go test -v -count=3 ./...
```

## Добавление новых тестов

### 1. Создайте новый тест-кейс

```go
func TestNewFeature(t *testing.T) {
    // Arrange
    // Act
    // Assert
}
```

### 2. Добавьте тест в таблицу параметров (для табличных тестов)

```go
func TestAPIHandler(t *testing.T) {
    tests := []struct {
        name     string
        // ... параметры
    }{
        {
            name: "test case 1",
            // ... параметры
        },
        // ... другие тест кейсы
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // тестовая логика
        })
    }
}
```

### 3. Добавьте бенчмарк

```go
func BenchmarkNewFeature(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // код для бенчмарка
    }
}
```

## Рекомендации

1. **Регулярно запускайте тесты** перед каждым коммитом
2. **Используйте покрытие кода** для выявления непроверенных участков
3. **Следите за производительностью** с помощью бенчмарков
4. **Добавляйте тесты** для новых функций сразу после их реализации
5. **Используйте табличные тесты** для тестирования различных сценариев

## Дополнительные команды

### Очистка кэша тестов

```bash
go clean -testcache
```

### Запуск тестов с выводом в файл

```bash
go test -v ./... > test_output.txt 2>&1
```

### Просмотр результатов тестов в реальном времени

```bash
gotest.tools/gotestsum --format testname -- ./...
```

## Контакты

При возникновении проблем с автотестами обратитесь к основной документации проекта или создайте issue в репозитории.
