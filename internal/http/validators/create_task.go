package validators

import (
	"net/http"

	"github.com/labstack/echo/v4"

	dto "task-pool-system.com/task-pool-system/internal/data_models"
)

func ValidateCreateTaskRequest(d *dto.TaskRequestData) error {
	if d.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if d.Description == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "description is required")
	}
	return nil
}
