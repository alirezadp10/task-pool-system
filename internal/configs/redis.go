package config

import (
	"log"

	"github.com/redis/rueidis"
)

func NewRedisClient(addr string) rueidis.Client {
	redisClient, err := rueidis.NewClient(
		rueidis.ClientOption{
			InitAddress: []string{addr},
		},
	)
	if err != nil {
		log.Fatalf("failed to create redis client: %v", err)
	}

	return redisClient
}
