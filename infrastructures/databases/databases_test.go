package databases

import (
	"context"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
)

type DatabasesSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *DatabasesSuite) SetupSuite() {
	suite.ctx = context.Background()
}

func TestDatabasesSuite(t *testing.T) {
	suite.Run(t, new(DatabasesSuite))
}

func (suite *DatabasesSuite) TestConnectRedis() {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.NoError(err)
	defer func() { _ = redisContainer.Terminate(ctx) }()

	env := env.RedisEnv{
		RedisAddress:  "localhost:6379",
		RedisPassword: "",
		RedisDb:       0,
	}

	redisFactory := NewRedisFactory(env)
	redisClient := redisFactory.ConnectRedis()
	suite.NotNil(redisClient)
}

func (suite *DatabasesSuite) TestConnectElasticsearch() {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.17.4",
		ExposedPorts: []string{"9200/tcp"},
		WaitingFor:   wait.ForListeningPort("9200/tcp"),
	}
	elasticsearchContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.NoError(err)
	defer func() { _ = elasticsearchContainer.Terminate(ctx) }()

	env := env.ElasticsearchEnv{
		ElasticsearchAddress: "http://localhost:9200",
	}

	elasticsearchFactory := NewElasticsearchFactory(env)
	elasticsearchClient, err := elasticsearchFactory.ConnectElasticsearch()
	suite.NotNil(elasticsearchClient)
	suite.NoError(err)
}
