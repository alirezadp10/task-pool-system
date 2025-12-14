# ---------- Build stage ----------
FROM golang:1.24-alpine AS build

WORKDIR /src

# Install build dependencies (cached unless Dockerfile changes)
RUN apk add --no-cache git build-base

# Copy only go.mod and go.sum first (maximizes cache reuse)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build application with CGO enabled (SQLite)
RUN CGO_ENABLED=1 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/app .

# ---------- Runtime stage ----------
FROM alpine:3.20

# Install runtime SQLite libs (cached)
RUN apk add --no-cache sqlite-libs

# Create non-root user
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Create data directory for SQLite and set permissions
RUN mkdir -p /app/data && chown -R app:app /app

# Copy binary from build stage
COPY --from=build /out/app /usr/local/bin/app

USER app

# Documentation-only (actual port comes from APP_URL)
EXPOSE 9000

# Cobra subcommand
ENTRYPOINT ["app", "server"]
