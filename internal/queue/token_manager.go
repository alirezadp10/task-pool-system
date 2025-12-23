package queue

import (
	"context"
	"errors"
)

type TokenManager interface {
	AcquireToken(ctx context.Context) error

	ReleaseToken(ctx context.Context) error

	InitializeTokens(ctx context.Context, count int) error
}

var ErrNoTokenAvailable = errors.New("no queue token available")
