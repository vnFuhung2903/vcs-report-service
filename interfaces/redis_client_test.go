package interfaces

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-report-service/entities"
)

type RedisClientSuite struct {
	suite.Suite
	miniRedis   *miniredis.Miniredis
	redisClient *redis.Client
	client      IRedisClient
}

func (s *RedisClientSuite) SetupTest() {
	var err error
	s.miniRedis, err = miniredis.Run()
	s.Require().NoError(err)

	opt := &redis.Options{
		Addr: s.miniRedis.Addr(),
	}
	s.redisClient = redis.NewClient(opt)
	s.client = NewRedisClient(s.redisClient)
}

func (s *RedisClientSuite) TearDownTest() {
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.miniRedis != nil {
		s.miniRedis.Close()
	}
}

func TestRedisClientSuite(t *testing.T) {
	suite.Run(t, new(RedisClientSuite))
}

func (s *RedisClientSuite) TestGetNonExistentKey() {
	result, err := s.client.Get(context.Background(), "non-existent-key")

	s.NoError(err)
	s.Empty(result)
	s.IsType([]entities.ContainerWithStatus{}, result)
}

func (s *RedisClientSuite) TestGet() {
	testData := []entities.ContainerWithStatus{
		{
			ContainerId: "container-1",
			Status:      entities.ContainerOn,
		},
		{
			ContainerId: "container-2",
			Status:      entities.ContainerOff,
		},
	}

	jsonData, err := json.Marshal(testData)
	s.Require().NoError(err)

	testKey := "test-containers"
	err = s.redisClient.Set(context.Background(), testKey, string(jsonData), 0).Err()
	s.Require().NoError(err)

	result, err := s.client.Get(context.Background(), testKey)

	s.NoError(err)
	s.Len(result, 2)
	s.Equal(testData[0].ContainerId, result[0].ContainerId)
	s.Equal(testData[0].Status, result[0].Status)
	s.Equal(testData[1].ContainerId, result[1].ContainerId)
	s.Equal(testData[1].Status, result[1].Status)
}

func (s *RedisClientSuite) TestGetInvalidJSON() {
	testKey := "test-invalid-json"
	invalidJSON := "{invalid json data"
	err := s.redisClient.Set(context.Background(), testKey, invalidJSON, 0).Err()
	s.Require().NoError(err)

	result, err := s.client.Get(context.Background(), testKey)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "invalid character")
}

func (s *RedisClientSuite) TestGetContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := s.client.Get(ctx, "any-key")

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "context canceled")
}
