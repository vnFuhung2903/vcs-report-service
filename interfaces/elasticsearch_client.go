package interfaces

import (
	"context"

	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
)

type IElasticsearchClient interface {
	Do(ctx context.Context, req esapi.Request) (*esapi.Response, error)
}

type ElasticsearchClient struct {
	client *elasticsearch.Client
}

func NewElasticsearchClient(client *elasticsearch.Client) *ElasticsearchClient {
	return &ElasticsearchClient{client: client}
}

func (c *ElasticsearchClient) Do(ctx context.Context, req esapi.Request) (*esapi.Response, error) {
	return req.Do(ctx, c.client)
}
