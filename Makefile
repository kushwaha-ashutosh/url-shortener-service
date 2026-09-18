.PHONY: up up-deps down run test build

# One-command demo: builds and runs Postgres, Redis, and the app itself.
up:
	docker compose up -d --build

# Just the dependencies, for local dev against `make run` with faster
# edit/rebuild cycles than rebuilding the app's Docker image each time.
up-deps:
	docker compose up -d postgres redis

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
