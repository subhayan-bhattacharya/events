# Local settings (DB_URL, POSTGRES_PASSWORD) come from the git-ignored ../.env.
# Copy ../.env.example to ../.env to get started. Production uses Kubernetes secrets.
-include ../.env
export

ifndef DB_URL
$(error DB_URL is not set: copy ../.env.example to ../.env)
endif
GOOSE  := go tool goose -dir db/migrations

.PHONY: db-up migrate-up migrate-down migrate-new sqlc test run

db-up:        ## start local Postgres
	docker compose -f ../docker-compose.yml up -d db

migrate-up:   ## apply all pending migrations
	$(GOOSE) postgres "$(DB_URL)" up

migrate-down: ## undo the most recent migration
	$(GOOSE) postgres "$(DB_URL)" down

migrate-new:  ## create a migration: make migrate-new name=users_sessions
	@test -n "$(name)" || (echo "usage: make migrate-new name=<name>" && exit 1)
	$(GOOSE) create $(name) sql -s

sqlc:         ## generate Go code from db/queries
	go tool sqlc generate

test:
	go test ./...

run:
	DB_URL="$(DB_URL)" go run ./cmd/api
