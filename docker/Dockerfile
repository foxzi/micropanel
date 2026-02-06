# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ files
RUN templ generate

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -o micropanel ./cmd/micropanel

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

# Copy binary
COPY --from=builder /app/micropanel .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy static files
COPY --from=builder /app/web ./web

# Create directories
RUN mkdir -p /var/www/panel/sites /app/data

EXPOSE 8080

CMD ["./micropanel"]
