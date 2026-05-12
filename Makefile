APP_NAME := taxi-api
MIGRATIONS_DIR := migrations
DATABASE_URL ?= postgres://taxi:taxi_password@localhost:5432/taxi?sslmode=disable

.PHONY: run build test tidy fmt lint docker-up docker-down migrate-up migrate-down migrate-create swagger

run:
	go run ./cmd/api

build:
	go build -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './docs/*')
	go run golang.org/x/tools/cmd/goimports@v0.33.0 -w .

lint:
	golangci-lint run ./...

docker-up:
	docker compose up --build

docker-down:
	docker compose down

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g ./cmd/api/main.go -o ./docs --parseDependency --parseInternal
