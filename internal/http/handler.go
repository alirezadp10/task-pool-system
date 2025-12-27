package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	dto "task-pool-system.com/task-pool-system/internal/data_models"
	exception "task-pool-system.com/task-pool-system/internal/exceptions"
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
	var req dto.TaskRequestData
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(exception.ErrInvalidJSON.StatusCode, exception.ErrInvalidJSON.Error())
	}
	if err := validators.ValidateCreateTaskRequest(&req); err != nil {
		return err
	}

	task, err := h.taskService.CreateTask(c.Request().Context(), req.Title, req.Description)
	if err != nil {
		return echo.NewHTTPError(exception.StatusCode(err), err.Error())
	}

	return c.JSON(http.StatusAccepted, task)
}

func (h *Handler) GetTask(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(exception.ErrTaskIDRequired.StatusCode, exception.ErrTaskIDRequired.Error())
	}

	task, err := h.taskService.GetTask(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(exception.StatusCode(err), err.Error())
	}

	return c.JSON(http.StatusOK, task)
}

func (h *Handler) ListTasks(c echo.Context) error {
	tasks, err := h.taskService.ListTasks(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(exception.StatusCode(err), err.Error())
	}

	return c.JSON(http.StatusOK, echo.Map{
		"count": len(tasks),
		"tasks": tasks,
	})
}
