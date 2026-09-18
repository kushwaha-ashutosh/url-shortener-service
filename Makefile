.PHONY: up down run test build

up:
	docker compose up -d

down:
	docker compose down

run:
	go run ./cmd/server

test:
	go test ./... -v

loadtest:
	docker run --rm -v "$$(pwd)/loadtest:/scripts" -e BASE_URL=http://host.docker.internal:8081 grafana/k6 run /scripts/redirect.js

build:
	go build -o bin/server ./cmd/server
