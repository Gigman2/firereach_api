include .env
export

.PHONY: run dev build test migrate-up migrate-down generate docker-up docker-down seed

run:
	go run ./cmd/api

dev:
	nodemon --watch './**/*.go' --signal SIGTERM --exec 'go run ./cmd/api' --ext go

build:
	go build -o bin/firereach-api ./cmd/api

test:
	go test ./...

migrate-up:
	$(HOME)/go/bin/migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(HOME)/go/bin/migrate -path migrations -database "$(DATABASE_URL)" down

generate:
	sqlc generate

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

tidy:
	go mod tidy

seed:
	docker compose exec -T db psql -U firereach -d firereach < seeds/dev_stations.sql
