package cmd

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"
	"log"
	"os/signal"
	"syscall"
	middleware "task-pool-system.com/task-pool-system/internal/http/middlewares"
	"time"

	config "task-pool-system.com/task-pool-system/internal/configs"
	httpapi "task-pool-system.com/task-pool-system/internal/http"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
	"task-pool-system.com/task-pool-system/internal/services"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long:  "Starts the task pool HTTP API and worker pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := godotenv.Load(); err != nil {
			log.Println(".env file not found, using environment variables")
		}

		cfg := config.Load()
		database := config.NewDatabaseClient(cfg.DatabaseDSN)
		taskRepo := repository.NewTaskRepository(database)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		poolService := services.NewPoolService(ctx, taskRepo, cfg.Workers, cfg.QueueSize, cfg.PollIntervalSeconds, cfg.PollBatchSize)

		taskService := services.NewTaskService(taskRepo)

		e := echo.New()
		e.Use(middleware.RateLimiter(cfg.RateLimit, time.Minute))
		httpapi.NewTaskHandler(e, taskService)

		go func() {
			log.Printf("HTTP server listening on %s", cfg.AppURL)
			if err := e.Start(cfg.AppURL); err != nil {
				log.Printf("server stopped: %v", err)
			}
		}()

		<-ctx.Done()

		echoCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second)
		defer cancel()
		_ = e.Shutdown(echoCtx)

		poolService.Shutdown()

		log.Println("HTTP server and worker pool shut down gracefully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
