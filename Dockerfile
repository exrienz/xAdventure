# Development stage with hot reload
FROM golang:1.25-alpine AS dev

RUN apk add --no-cache gcc musl-dev git
RUN go install github.com/air-verse/air@v1.62.0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8001

CMD ["/go/bin/air", "-c", ".air.toml"]

# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -buildvcs=false -o /server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app
COPY --from=builder /server /app/server
COPY --from=builder /app/web /app/web
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8001

CMD ["./server"]
