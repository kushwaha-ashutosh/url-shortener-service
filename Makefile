.PHONY: up down run test build

up:
	docker compose up -d

down:
	docker compose down

run:
	go run ./cmd/server

test:
	go test ./... -v

build:
	go build -o bin/server ./cmd/server
