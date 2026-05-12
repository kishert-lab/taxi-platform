run:
	go run ./cmd/api

build:
	go build -o bin/taxi-api ./cmd/api

docker-up:
	docker compose up -d

docker-down:
	docker compose down

swagger:
	swag init -g ./cmd/api/main.go