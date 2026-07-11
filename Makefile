-include .env
export

.PHONY: run build test race vet fmt tidy bruno migrate-up migrate-down

run:
	go run ./cmd/identity-service

build:
	go build -o bin/identity-service ./cmd/identity-service

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

bruno:
	cd bruno && bru run --env Local

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1
