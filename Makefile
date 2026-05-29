.PHONY: run build test sqlc migrate-up migrate-down mockery

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

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
