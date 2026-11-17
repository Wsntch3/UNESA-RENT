DB_URL ?= postgres://unesarent:unesarent_dev@postgres:5432/unesarent?sslmode=disable

.PHONY: install backend frontend dev db-up db-down db-reset db-migrate db-migrate-down db-seed

install:
	cd backend && go mod download
	cd frontend && npm install

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

dev:
	@trap 'kill 0' EXIT INT TERM; \
	(cd backend && go run ./cmd/server) & \
	(cd frontend && npm run dev) & \
	wait

db-up:
	cd postgres && docker compose up -d

db-down:
	cd postgres && docker compose down

db-reset:
	cd postgres && docker compose down -v && docker compose up -d

db-migrate:
	cd postgres && docker compose run --rm migrate

db-migrate-down:
	cd postgres && docker compose run --rm migrate -path=/migrations -database="$(DB_URL)" down 1

db-seed:
	cd postgres && docker compose exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v ON_ERROR_STOP=1' < seed/dev_seed.sql
