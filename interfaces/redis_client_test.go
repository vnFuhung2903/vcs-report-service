package interfaces

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisClient(t *testing.T) {
	opt := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	rds := redis.NewClient(opt)
	assert.NotNil(t, rds)

	redisClient := NewRedisClient(rds)

	_, err := redisClient.Get(context.Background(), "test-key")
	assert.Error(t, err)
}
