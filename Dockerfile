# 1. stage: build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# go mod dosyaları
COPY go.mod go.sum ./
RUN go mod download

# tüm proje
COPY . .

# statik binary (CGO kapalı, minimal)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o event-metrics-service ./cmd/api

# 2. stage: runtime
FROM alpine:3.20

WORKDIR /app

# küçük runtime dependency'ler (opsiyonel)
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/event-metrics-service /app/event-metrics-service

# Migrations'ları da image içine koymak istersen:
COPY migrations /app/migrations

ENV POSTGRES_DSN=postgres://user:password@postgres:5432/eventdb?sslmode=disable

EXPOSE 8080

CMD ["/app/event-metrics-service"]
