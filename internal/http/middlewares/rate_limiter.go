package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

func RateLimiter(limit int, window time.Duration) echo.MiddlewareFunc {
	type bucket struct {
		count int
		start time.Time
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			now := time.Now()
			key := c.RealIP()

			mu.Lock()
			b, ok := buckets[key]
			if !ok || now.Sub(b.start) > window {
				b = &bucket{start: now}
				buckets[key] = b
			}

			if b.count >= limit {
				mu.Unlock()
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}

			b.count++
			mu.Unlock()

			return next(c)
		}
	}
}
