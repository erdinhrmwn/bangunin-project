.PHONY: run run-worker build migrate-up migrate-down migrate-create seed test lint gen-proto

run:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

migrate-up:
	go run ./cmd/migrate -cmd up

migrate-down:
	go run ./cmd/migrate -cmd down

migrate-create:
	go run ./cmd/migrate -cmd create -name $(name)

seed:
	go run ./cmd/migrate -cmd up -seed

test:
	go test ./...

lint:
	golangci-lint run ./...

gen-proto:
	@echo "gen-proto: no .proto files yet (added in Fase 5)"
