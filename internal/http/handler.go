package http

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	dto "task-pool-system.com/task-pool-system/internal/data_models"
	"task-pool-system.com/task-pool-system/internal/http/validators"
	"task-pool-system.com/task-pool-system/internal/services"
)

type Handler struct {
	taskService *services.TaskService
}

func NewHandler(taskService *services.TaskService) *Handler {
	return &Handler{
		taskService: taskService,
	}
}

func (h *Handler) CreateTask(c echo.Context) error {
	var req dto.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON payload")
	}
	if err := validators.ValidateCreateTaskRequest(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	task, err := h.taskService.CreateTask(ctx, req.Title, req.Description)
	if err != nil {
		if errors.Is(err, services.ErrTaskQueueFull) {
			return echo.NewHTTPError(
				http.StatusTooManyRequests,
				"task queue is full",
			)
		}

		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task")
	}

	return c.JSON(http.StatusAccepted, task)
}

func (h *Handler) GetTask(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "task id is required")
	}

	task, err := h.taskService.GetTask(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "task not found")
	}

	return c.JSON(http.StatusOK, task)
}

func (h *Handler) ListTasks(c echo.Context) error {
	tasks, err := h.taskService.ListTasks(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list tasks")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"count": len(tasks),
		"tasks": tasks,
	})
}
