.PHONY: up down build run sqlc migrate-up migrate-down lint

up:
	docker compose up --build -d

down:
	docker compose down

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

sqlc:
	sqlc generate

migrate-up:
	migrate -path db/migrations -database "postgres://postgres:postgres@localhost:5432/usermanagement?sslmode=disable" up

migrate-down:
	migrate -path db/migrations -database "postgres://postgres:postgres@localhost:5432/usermanagement?sslmode=disable" down

lint:
	golangci-lint run ./...
