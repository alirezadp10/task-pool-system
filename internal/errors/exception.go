package errors

import (
	"errors"
	"net/http"
)

type Exception struct {
	Message    string
	StatusCode int
}

func (e *Exception) Error() string {
	return e.Message
}

func StatusCode(err error) int {
	var appErr *Exception
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}
