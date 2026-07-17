.PHONY: build test run lint clean docker-build docker-up docker-down

build:
	go build -o server ./cmd/server

test:
	go test -race ./...

run:
	go run ./cmd/server

lint:
	golangci-lint run ./...

clean:
	rm -f server
	docker compose down -v

docker-build:
	docker compose build

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
