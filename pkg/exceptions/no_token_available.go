package exceptions

import "net/http"

var ErrNoTokenAvailable = &Exception{
	Message:    "no queue token available",
	StatusCode: http.StatusServiceUnavailable,
}
