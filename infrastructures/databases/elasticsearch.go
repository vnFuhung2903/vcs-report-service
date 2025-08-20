package databases

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
)

type IElasticsearchFactory interface {
	ConnectElasticsearch() (*elasticsearch.Client, error)
}

type elasticsearchFactory struct {
	address string
}

func NewElasticsearchFactory(env env.ElasticsearchEnv) IElasticsearchFactory {
	return &elasticsearchFactory{address: env.ElasticsearchAddress}
}

func (f *elasticsearchFactory) ConnectElasticsearch() (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{f.address},
	}
	es, err := elasticsearch.NewClient(cfg)
	return es, err
}
