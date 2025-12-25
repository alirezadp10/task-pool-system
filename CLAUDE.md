# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A concurrent task pool system in Go that processes tasks asynchronously using a fixed-size worker pool. Tasks are submitted via REST API, persisted in SQLite, and processed with optimistic locking to prevent race conditions.

**Tech Stack:** Go 1.24, Echo (HTTP), Cobra (CLI), GORM (ORM), SQLite (persistence), Redis (rate limiting)

## Development Commands

### Building
```bash
go build -o task-pool .
```

### Running Tests
```bash
make test
# Or with race detection:
go test -v ./... -race
```

### Running Locally
```bash
# Copy environment configuration
cp .env.example .env

# Start the server (requires Redis running locally)
go run main.go server
```

### Running with Docker
```bash
# Builds app and starts with Redis
docker compose up --build
```

## Architecture

### Core Components

**PoolService** (`internal/services/pool_service.go`)
- Manages worker goroutines and task queue channel
- Worker pool initialized at startup with fixed number of workers
- Each worker continuously pulls task IDs from `queue chan string`
- **Polling Publisher (Transactional Outbox Pattern):**
  - Periodic poller goroutine fetches pending tasks from database
  - Polls every `TASK_POLL_INTERVAL_SECONDS` for up to `TASK_POLL_BATCH_SIZE` tasks
  - Tasks are durably persisted before being queued for processing
  - Query: `WHERE status = 'pending' AND started_at IS NULL ORDER BY created_at ASC`
  - Handles queue-full gracefully (stops enqueuing, retries next poll)
- Implements graceful shutdown via channel closure and WaitGroup
- Task processing: fetch → mark in_progress → simulate work (1-5s) → mark completed

**TaskService** (`internal/services/task_service.go`)
- Thin wrapper around TaskRepository for business logic
- Handles task CRUD operations

**TaskRepository** (`internal/repositories/task_repository.go`)
- GORM-based data access layer
- **Optimistic locking:** Uses `version` field incremented on each update
- `Update()` checks `WHERE id = ? AND version = ?` and returns `ErrOptimisticLock` if no rows affected
- This prevents concurrent workers from processing the same task

**HTTP Layer** (`internal/http/`)
- `handler.go`: Echo handlers for task endpoints
- `routes.go`: Route registration with rate limiting middleware
- `middlewares/rate_limiter.go`: In-memory rate limiter using sync.Mutex and time windows

### Project Structure

```
cmd/
  server.go       - Server startup, dependency injection, graceful shutdown
  root.go         - Cobra root command
internal/
  configs/        - Environment variable loading and validation
  constants/      - Task status constants (pending, in_progress, completed, failed)
  data_models/    - DTOs for HTTP requests/responses
  errors/         - Custom error types with HTTP status codes
  http/           - HTTP handlers, routes, validators, middlewares
  models/         - GORM database models
  repositories/   - Data access layer
  services/       - Business logic (PoolService, TaskService)
```

### Key Patterns

**Optimistic Locking**
- Task model has `Version uint` field
- On update: `WHERE id = ? AND version = ?` then `SET version = version + 1`
- Prevents race conditions when multiple workers try updating the same task

**Graceful Shutdown** (`cmd/server.go:57-70`)
- Listens for SIGINT/SIGTERM
- Creates context with timeout (`SHUTDOWN_TIMEOUT_SECONDS`)
- Calls `e.Shutdown(ctx)` for HTTP server
- Calls `poolService.Shutdown(ctx)` which closes queue channel and waits for workers

**Worker Pool Pattern**
- Fixed number of goroutines created at startup
- Workers block on `for taskID := range p.queue`
- Queue is buffered channel sized by `TASK_QUEUE_SIZE`
- Channel closure signals workers to exit

**Error Handling**
- Custom `Exception` type in `internal/errors/` with StatusCode field
- `StatusCode(err)` helper extracts HTTP status or defaults to 500
- Named errors: `ErrTaskNotFound`, `ErrOptimisticLock`, `ErrInvalidLimit`, etc.

### Configuration

All config via environment variables loaded in `internal/configs/configs.go`:
- Validation happens at startup (fails fast if invalid)
- See `.env.example` for all required variables
- Docker Compose overrides `REDIS_HOST` to use service name

**Key Polling Configuration:**
- `TASK_POLL_INTERVAL_SECONDS`: How often to poll for pending tasks (default: 5)
- `TASK_POLL_BATCH_SIZE`: Max tasks to fetch per poll (default: 10)

**Important:** Current implementation is designed for **single-instance deployment**. For horizontal scaling (multiple instances), you would need to add distributed locking (e.g., Redis-based) to the polling mechanism to prevent duplicate processing.

## Testing Notes

- Tests use the race detector (`-race` flag)
- PoolService tests verify concurrent task submission and worker processing
- No test database setup required (tests likely use in-memory SQLite or mocks)