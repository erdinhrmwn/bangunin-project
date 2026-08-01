.PHONY: help run run-worker build clean migrate-up migrate-down migrate-create seed \
        test test-cover lint fmt vet tidy gen-proto \
        docker-up docker-down docker-logs docker-build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## App
run: ## Run the API server
	@echo "🚀 starting API server..."
	go run ./cmd/api

run-worker: ## Run the Asynq worker
	@echo "⚙️  starting worker..."
	go run ./cmd/worker

build: ## Build API + worker binaries
	@echo "🔨 building binaries..."
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	@echo "✅ build done -> bin/api, bin/worker"

clean: ## Remove build artifacts
	@echo "🧹 cleaning bin/..."
	rm -rf bin/

## Database
migrate-up: ## Apply all up migrations
	@echo "⬆️  applying migrations..."
	go run ./cmd/migrate -cmd up
	@echo "✅ migrations up"

migrate-down: ## Roll back the last migration
	@echo "⬇️  rolling back migration..."
	go run ./cmd/migrate -cmd down
	@echo "✅ migration rolled back"

migrate-create: ## Create a migration pair (make migrate-create name=xxx)
	@echo "📝 creating migration: $(name)"
	go run ./cmd/migrate -cmd create -name $(name)

seed: ## Migrate up + run seeders
	@echo "🌱 seeding database..."
	go run ./cmd/migrate -cmd up -seed
	@echo "✅ seed done"

## Quality
test: ## Run tests
	@echo "🧪 running tests..."
	go test ./...

test-cover: ## Run tests with coverage report
	@echo "🧪 running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint: ## Run golangci-lint
	@echo "🔍 linting..."
	golangci-lint run ./...

fmt: ## Format code (gofmt + goimports + goimports-reviser)
	@echo "🎨 formatting code..."
	gofmt -l -w .
	@command -v goimports >/dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -l -w .
	@command -v goimports-reviser >/dev/null || go install github.com/incu6us/goimports-reviser/v3@latest
	goimports-reviser -rm-unused -project-name erdinhrmwn/bangunin -separate-named -format -recursive .

vet: ## Run go vet
	@echo "🔎 vetting code..."
	go vet ./...

tidy: ## Tidy go.mod/go.sum
	@echo "📦 tidying go modules..."
	go mod tidy

gen-proto: ## Generate gRPC code from .proto files
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/payment/v1/payment.proto \
		proto/notification/v1/notification.proto

## Docker
docker-up: ## Start local infra (postgres, redis, minio)
	@echo "🐳 starting local infra..."
	docker compose -f deploy/docker-compose.yml up -d

docker-down: ## Stop local infra
	@echo "🐳 stopping local infra..."
	docker compose -f deploy/docker-compose.yml down

docker-logs: ## Tail local infra logs
	docker compose -f deploy/docker-compose.yml logs -f

docker-build: ## Build api/worker images
	@echo "🐳 building docker images..."
	docker compose -f deploy/docker-compose.yml build
