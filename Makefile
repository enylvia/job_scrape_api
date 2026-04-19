.PHONY: db-up db-down db-reset migrate seed seed-sources db-setup api worker e2e-worker preview test

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-reset:
	docker compose down -v

migrate:
	go run ./cmd/dbsetup -seed=false

seed:
	go run ./cmd/dbsetup -migrate=false

seed-sources: seed

db-setup:
	go run ./cmd/dbsetup

api:
	go run ./cmd/api

worker:
	go run ./cmd/worker

e2e-worker: db-setup
	go run ./cmd/worker

e2e-worker: export DB_ENABLED=true
e2e-worker: export DB_HOST=127.0.0.1

preview:
	go run ./cmd/collectorpreview

test:
	go test ./...
