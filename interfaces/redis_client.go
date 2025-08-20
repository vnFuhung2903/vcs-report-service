package interfaces

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type IRedisClient interface {
	Get(ctx context.Context, key string) ([]string, error)
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(client *redis.Client) IRedisClient {
	return &RedisClient{client: client}
}

func (c *RedisClient) Get(ctx context.Context, key string) ([]string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []string{}, nil
	} else if err != nil {
		return nil, err
	}

	var result []string
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return result, nil
}
