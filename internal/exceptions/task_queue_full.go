package exceptions

import "net/http"

var ErrTaskQueueFull = &Exception{
	Message:    "task queue is full",
	StatusCode: http.StatusTooManyRequests,
}
