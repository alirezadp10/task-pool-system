package http

import (
	"time"

	"github.com/labstack/echo/v4"

	middleware "task-pool-system.com/task-pool-system/internal/http/middlewares"
)

func Register(e *echo.Echo, h *Handler, rateLimitPerMinute int) {
	e.Use(middleware.RateLimiter(rateLimitPerMinute, time.Minute))

	e.POST("/tasks", h.CreateTask)
	e.GET("/tasks/:id", h.GetTask)
	e.GET("/tasks", h.ListTasks)
}
