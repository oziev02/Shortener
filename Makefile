.PHONY: run run-redis build clean docker-up docker-down deps fmt fmt-check lint vet check

# Запуск сервера (использует .env файл или переменные окружения)
run:
	go run cmd/server/main.go

# Запуск сервера с Redis (использует .env файл или переменные окружения)
run-redis:
	go run cmd/server/main.go -enable-redis

# Сборка бинарника
build:
	go build -o bin/shortener cmd/server/main.go

# Запуск Docker Compose
docker-up:
	docker-compose up -d

# Остановка Docker Compose
docker-down:
	docker-compose down

# Очистка
clean:
	rm -rf bin/
	go clean

# Установка зависимостей
deps:
	go mod download
	go mod tidy

# Форматирование кода и сортировка импортов
fmt:
	goimports -w .
	go fmt ./...

# Проверка форматирования (без изменений файлов)
fmt-check:
	@test -z $$(goimports -d . | head -n -1) || (echo "Code is not formatted. Run 'make fmt' to fix." && exit 1)
	@test -z $$(gofmt -d . | head -n -1) || (echo "Code is not formatted. Run 'make fmt' to fix." && exit 1)

# Запуск линтера
lint:
	golangci-lint run ./...

# Запуск встроенного анализатора
vet:
	go vet ./...

# Все проверки сразу
check: fmt-check vet lint
