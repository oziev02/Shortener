.PHONY: run run-redis build test clean docker-up docker-down deps

# Запуск сервера (использует .env файл или переменные окружения)
run:
	go run cmd/server/main.go

# Запуск сервера с Redis (использует .env файл или переменные окружения)
run-redis:
	go run cmd/server/main.go -enable-redis

# Сборка бинарника
build:
	go build -o bin/shortener cmd/server/main.go

# Запуск тестов
test:
	go test ./...

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

