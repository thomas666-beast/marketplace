DB_URL := postgres://marketplace:marketplace_dev@localhost:5432/marketplace?sslmode=disable

.PHONY: migrate-up migrate-down migrate-create

migrate-up:
	migrate -path internal/postgres/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path internal/postgres/migrations -database "$(DB_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/postgres/migrations -seq $$name
