package databases

import (
	"github.com/redis/go-redis/v9"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
)

type IRedisFactory interface {
	ConnectRedis() *redis.Client
}

type RedisFactory struct {
	Address  string
	Password string
	Db       int
}

func NewRedisFactory(env env.RedisEnv) IRedisFactory {
	return &RedisFactory{
		Address:  env.RedisAddress,
		Password: env.RedisPassword,
		Db:       env.RedisDb,
	}
}

func (f *RedisFactory) ConnectRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     f.Address,
		Password: f.Password,
		DB:       f.Db,
	})
}
