include app/cmd/.env
MIGRATION_DIR ?= infra/db/migrations
.PHONY: smithy-build
smithy-build:
	@echo "Building smithy model"
	@smithy build
	@go run build_specs.go