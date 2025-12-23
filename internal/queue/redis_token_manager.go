package queue

import (
	"context"

	"github.com/redis/rueidis"
)

type RedisTokenManager struct {
	client rueidis.Client
	key    string
}

func NewRedisTokenManager(client rueidis.Client, queueKey string) *RedisTokenManager {
	return &RedisTokenManager{
		client: client,
		key:    queueKey,
	}
}

func (r *RedisTokenManager) AcquireToken(ctx context.Context) error {
	cmd := r.client.B().Lpop().Key(r.key).Build()
	result := r.client.Do(ctx, cmd)

	if err := result.Error(); err != nil {
		if rueidis.IsRedisNil(err) {
			return ErrNoTokenAvailable
		}
		return err
	}

	return nil
}

func (r *RedisTokenManager) ReleaseToken(ctx context.Context) error {
	cmd := r.client.B().Rpush().Key(r.key).Element("1").Build()
	return r.client.Do(ctx, cmd).Error()
}

func (r *RedisTokenManager) InitializeTokens(ctx context.Context, count int) error {
	delCmd := r.client.B().Del().Key(r.key).Build()
	if err := r.client.Do(ctx, delCmd).Error(); err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		if err := r.ReleaseToken(ctx); err != nil {
			return err
		}
	}

	return nil
}
