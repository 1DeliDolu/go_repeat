# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build web server
RUN CGO_ENABLED=0 GOOS=linux go build -o web ./cmd/web

# Build worker
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/web /app/web
COPY --from=builder /app/worker /app/worker
COPY --from=builder /app/migrations ./migrations

# Make binaries executable
RUN chmod +x /app/web /app/worker

EXPOSE 8080

CMD ["/app/web"]
