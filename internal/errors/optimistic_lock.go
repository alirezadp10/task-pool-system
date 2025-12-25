package errors

import "net/http"

var ErrOptimisticLock = &Exception{
	Message:    "optimistic locking conflict",
	StatusCode: http.StatusConflict,
}
