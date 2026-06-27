# URL Shortner - Go

A URL shortener built in Go using a layered architecture (handler → service → repository) with PostgreSQL for persistence.

## Stack

- **Go** 1.26 — `net/http` stdlib (no framework)
- **PostgreSQL** 16 — via Docker Compose
- **pgx/v5** — PostgreSQL driver

## Project structure

```
├── main.go           # wiring, config, server startup
├── handler/          # HTTP layer — request decoding, response encoding
├── service/          # business logic — code generation, validation
└── repository/       # data layer — all SQL and database access
```

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)

## Setup

**1. Start the database**

```bash
docker compose up -d
```

**2. Run the server**

```bash
go run .
```

The server starts on `http://localhost:8080`.

## API

**Shorten a URL**

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

```json
{"short_code":"aB3xYz9","short_url":"http://localhost:8080/aB3xYz9"}
```

**Follow a short URL**

```bash
curl -L http://localhost:8080/aB3xYz9
```

Returns a `302` redirect to the original URL.

## Configuration

The server is configured via environment variables. All have defaults that work with the provided `docker-compose.yml`.

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://shortener:shortener@localhost:5432/shortener` | PostgreSQL connection string |
| `BASE_URL` | `http://localhost:8080` | Prefix used when building short URLs |
| `ADDR` | `:8080` | Address the server listens on |

## Build

```bash
go build -o go-url-shortner .
./go-url-shortner
```

## Test

```bash
go test ./...
```
