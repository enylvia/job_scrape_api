.PHONY: db-up db-down db-reset api test

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-reset:
	docker compose down -v

api:
	go run ./cmd/api

test:
	go test ./...
