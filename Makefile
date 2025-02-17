.PHONY: run test lint docker-dev docker-prod

run:
	go run ./cmd/app

tests:
	go test -v -cover -race ./...

lint:
	golangci-lint run

docker-dev:
		docker compose -f docker-compose.dev.yml down -v && docker compose -f docker-compose.dev.yml up --build -d

docker-prod:
		docker compose down -v && docker compose --env-file .env up --build -d