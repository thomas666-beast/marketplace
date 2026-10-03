migrate-up:
	migrate -path internal/postgres/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path internal/postgres/migrations -database "$(DB_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/postgres/migrations -seq $$name
