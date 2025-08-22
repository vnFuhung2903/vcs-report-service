package interfaces

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/vnFuhung2903/vcs-report-service/entities"
)

type IRedisClient interface {
	Get(ctx context.Context, key string) ([]entities.ContainerWithStatus, error)
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(client *redis.Client) IRedisClient {
	return &RedisClient{client: client}
}

func (c *RedisClient) Get(ctx context.Context, key string) ([]entities.ContainerWithStatus, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []entities.ContainerWithStatus{}, nil
	} else if err != nil {
		return nil, err
	}

	var result []entities.ContainerWithStatus
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return result, nil
}
