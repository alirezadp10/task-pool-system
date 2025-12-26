package validators

import (
	"net/http"

	"github.com/labstack/echo/v4"

	dto "task-pool-system.com/task-pool-system/internal/data_models"
)

func ValidateCreateTaskRequest(r *dto.TaskRequestData) error {
	if r.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if r.Description == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "description is required")
	}
	return nil
}
