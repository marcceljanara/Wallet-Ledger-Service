include .env
export

.PHONY: run build test sqlc migrate-up migrate-down mockery seed


run:
	go run ./cmd/server

seed:
	go run ./cmd/seed

build:
	go build -o bin/server.exe ./cmd/server

test:
	go test ./... -v -cover

sqlc:
	sqlc generate

mockery:
	mockery --all --dir=internal --output=internal/mocks --outpkg=mocks

migrate-up:
	migrate -path database/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path database/migrations -database "$(DATABASE_URL)" down
