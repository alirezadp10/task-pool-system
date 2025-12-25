package errors

import "net/http"

var ErrTaskIDRequired = &Exception{
	Message:    "task id is required",
	StatusCode: http.StatusBadRequest,
}
