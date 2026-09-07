include app/cmd/.env
MIGRATION_DIR ?= infra/db/migrations
.PHONY: migrate-up migrate-down migrate-new

migrate-up:
		@echo "Running up migrations..."
		@migrate -database "$(DB_URL)" -path $(MIGRATION_DIR) up

migrate-down:
		@echo "Rolling back last migration..."
		@migrate -database "$(DB_URL)" -path $(MIGRATION_DIR) down 1

migration-new:
		@read -p "Enter migration name: " name; \
		migrate create -ext sql -dir $(MIGRATION_DIR) -seq $$name