package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	config "task-pool-system.com/task-pool-system/internal/configs"
	sqlite "task-pool-system.com/task-pool-system/internal/configs"
	httpapi "task-pool-system.com/task-pool-system/internal/http"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
	"task-pool-system.com/task-pool-system/internal/services"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the HTTP API server",
	Long:  "Starts the task pool HTTP API and worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {

		if err := godotenv.Load(); err != nil {
			log.Println(".env file not found, using environment variables")
		}

		cfg := config.Load()

		redisClient := config.NewRedisClient(cfg.RedisAddr)
		defer redisClient.Close()

		{
			ctx := context.Background()

			if err := redisClient.Do(
				ctx,
				redisClient.B().Del().Key(cfg.RedisQueueKey).Build(),
			).Error(); err != nil {
				log.Fatalf("failed to reset redis queue tokens: %v", err)
			}

			if cfg.QueueSize > 0 {
				for i := 0; i < cfg.QueueSize; i++ {
					if err := redisClient.Do(
						ctx,
						redisClient.B().Rpush().Key(cfg.RedisQueueKey).Element("1").Build(),
					).Error(); err != nil {
						log.Fatalf("failed to initialize redis queue tokens: %v", err)
					}
				}
			}
		}

		sqlite := sqlite.New(cfg.DatabaseDSN)

		taskRepo := repository.NewTaskRepository(sqlite)

		pool := services.NewPoolService(
			taskRepo,
			cfg.Workers,
			cfg.QueueSize,
			redisClient,
			cfg.RedisQueueKey,
		)

		taskService := services.NewTaskService(taskRepo, pool, redisClient, cfg.RedisQueueKey)

		e := echo.New()

		handler := httpapi.NewHandler(taskService)
		httpapi.Register(e, handler, cfg.RateLimit)

		go func() {
			log.Printf("HTTP server listening on %s", cfg.AppURL)
			if err := e.Start(cfg.AppURL); err != nil {
				log.Printf("server stopped: %v", err)
			}
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		ctx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second,
		)
		defer cancel()

		_ = e.Shutdown(ctx)
		pool.Shutdown(ctx)

		log.Println("HTTP server and worker pool shut down gracefully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
