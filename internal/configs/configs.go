package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppURL                 string
	Workers                int
	QueueSize              int
	PollIntervalSeconds    int
	PollBatchSize          int
	DatabaseDSN            string
	RateLimit              int
	RedisAddr              string
	RedisQueueKey          string
	ShutdownTimeoutSeconds int
}

func Load() Config {
	appHost := getEnv("APP_HOST", "127.0.0.1")
	appPort := getEnv("APP_PORT", "8080")
	redisHost := getEnv("REDIS_HOST", "127.0.0.1")
	redisPort := getEnv("REDIS_PORT", "6379")

	cfg := Config{
		AppURL:                 fmt.Sprintf("%s:%s", appHost, appPort),
		Workers:                getEnvAsInt("TASK_WORKERS", 5),
		QueueSize:              getEnvAsInt("TASK_QUEUE_SIZE", 10),
		PollIntervalSeconds:    getEnvAsInt("TASK_POLL_INTERVAL_SECONDS", 5),
		PollBatchSize:          getEnvAsInt("TASK_POLL_BATCH_SIZE", 10),
		DatabaseDSN:            getEnv("DATABASE_DSN", "tasks.db"),
		RateLimit:              getEnvAsInt("RATE_LIMIT_PER_MINUTE", 60),
		RedisAddr:              fmt.Sprintf("%s:%s", redisHost, redisPort),
		RedisQueueKey:          getEnv("REDIS_QUEUE_KEY", "task_queue_tokens"),
		ShutdownTimeoutSeconds: getEnvAsInt("SHUTDOWN_TIMEOUT_SECONDS", 20),
	}

	validate(cfg)
	return cfg
}

func validate(cfg Config) {
	if cfg.AppURL == "" {
		log.Fatal("APP_URL must not be empty (e.g. 127.0.0.1:8080)")
	}
	if cfg.Workers <= 0 {
		log.Fatal("TASK_WORKERS must be greater than 0")
	}
	if cfg.QueueSize <= 0 {
		log.Fatal("TASK_QUEUE_SIZE must be greater than 0")
	}
	if cfg.PollIntervalSeconds <= 0 {
		log.Fatal("TASK_POLL_INTERVAL_SECONDS must be greater than 0")
	}
	if cfg.PollBatchSize <= 0 {
		log.Fatal("TASK_POLL_BATCH_SIZE must be greater than 0")
	}
	if cfg.DatabaseDSN == "" {
		log.Fatal("DATABASE_DSN must not be empty")
	}
	if cfg.RateLimit <= 0 {
		log.Fatal("RATE_LIMIT_PER_MINUTE must be greater than 0")
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid integer value for %s", key)
		}
		return i
	}
	return defaultVal
}
