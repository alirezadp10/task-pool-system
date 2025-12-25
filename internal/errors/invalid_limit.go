package errors

import "net/http"

var ErrInvalidLimit = &Exception{
	Message:    "limit must be positive",
	StatusCode: http.StatusBadRequest,
}
