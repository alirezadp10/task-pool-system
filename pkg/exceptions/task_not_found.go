package exceptions

import "net/http"

var ErrTaskNotFound = &Exception{
	Message:    "task not found",
	StatusCode: http.StatusNotFound,
}
