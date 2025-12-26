package exceptions

import "net/http"

var ErrInvalidJSON = &Exception{
	Message:    "invalid JSON payload",
	StatusCode: http.StatusBadRequest,
}
