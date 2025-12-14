# Task Pool System

A simple, concurrent **task pool system** implemented in Go.  
Clients can submit tasks via a REST API; tasks are processed asynchronously by a fixed-size worker pool. Task state is persisted using SQLite and exposed through HTTP endpoints.

This project demonstrates:

- Go concurrency (goroutines, channels)
- REST API design
- Worker pool pattern
- Graceful shutdown
- Redis-backed rate limiting and queue token management
- Dockerized deployment
- Clean, idiomatic Go project structure

## Tech Stack

- **Go** (1.24)
- **Echo** – HTTP framework
- **Cobra** – CLI
- **GORM** – ORM
- **SQLite** – persistence
- **Redis** – rate limiting and queue management
- **Docker / Docker Compose**

## Features

- Submit tasks asynchronously
- Query task status by ID
- List all tasks
- Fixed-size worker pool
- Simulated task processing time (1–5 seconds)
- Graceful shutdown on SIGINT / SIGTERM
- SQLite persistence with Docker volume
- Configurable via environment variables
- Redis-based resource management

## Configuration

The application is configured via environment variables.

### Required Variables

| Variable                    | Description                         | Example                |
|-----------------------------|-------------------------------------|------------------------|
| `APP_HOST`                  | HTTP server bind host               | `127.0.0.1`            |
| `APP_PORT`                  | HTTP server port                    | `8080`                 |
| `TASK_WORKERS`              | Number of worker goroutines         | `5`                    |
| `TASK_QUEUE_SIZE`           | Maximum pending tasks               | `10`                   |
| `DATABASE_DSN`              | SQLite database path                | `tasks.db`             |
| `RATE_LIMIT_PER_MINUTE`     | API rate limit                      | `60`                   |
| `REDIS_HOST`                | Redis server host                   | `127.0.0.1`            |
| `REDIS_PORT`                | Redis server port                   | `6379`                 |
| `REDIS_QUEUE_KEY`           | Redis key for task queue tokens     | `task_queue_tokens`    |
| `SHUTDOWN_TIMEOUT_SECONDS`  | Graceful shutdown timeout           | `20`                   |

## Running with Docker

Docker Compose will set up both the application and a Redis instance automatically.

### 1. Configure environment

Ensure your `.env` is set up correctly. For Docker, `REDIS_HOST` and `REDIS_PORT` are overridden by the `docker-compose.yml` to use the Redis service on the Docker network (`redis:6379`).

### 2. Build and start

```bash
docker compose up --build
```

## Running Tests

The project includes unit tests for both `TaskService` and `PoolService` (including concurrent submission and worker processing).

To run all tests:

```bash
make test
```

Or using `go test` directly:

```bash
go test -v ./... -race
```

## API Endpoints

### Create a Task

**POST** `/tasks`

```bash
curl -X POST http://127.0.0.1:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"example","description":"demo task"}'
```

### Get Task by ID
**GET** `/tasks/{id}`

```bash
curl http://127.0.0.1:8080/tasks/<task_id>
```

### List All Tasks
**GET** `/tasks`

```bash
curl http://127.0.0.1:8080/tasks
```
