package env

import (
	"errors"

	"github.com/spf13/viper"
)

type AuthEnv struct {
	JWTSecret string
}

type ElasticsearchEnv struct {
	ElasticsearchAddress string
}

type GomailEnv struct {
	MailUsername string
	MailPassword string
}

type RedisEnv struct {
	RedisAddress  string
	RedisPassword string
	RedisDb       int
}

type LoggerEnv struct {
	Level      string
	FilePath   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
}

type Env struct {
	AuthEnv          AuthEnv
	ElasticsearchEnv ElasticsearchEnv
	GomailEnv        GomailEnv
	RedisEnv         RedisEnv
	LoggerEnv        LoggerEnv
}

func LoadEnv() (*Env, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("ELASTICSEARCH_ADDRESS", "http://localhost:9200")
	v.SetDefault("REDIS_ADDRESS", "localhost:6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("ZAP_LEVEL", "info")
	v.SetDefault("ZAP_FILEPATH", "./logs/app.log")
	v.SetDefault("ZAP_MAXSIZE", 100)
	v.SetDefault("ZAP_MAXAGE", 10)
	v.SetDefault("ZAP_MAXBACKUPS", 30)

	authEnv := AuthEnv{
		JWTSecret: v.GetString("JWT_SECRET_KEY"),
	}
	if authEnv.JWTSecret == "" {
		return nil, errors.New("auth environment variables are empty")
	}

	elasticsearchEnv := ElasticsearchEnv{
		ElasticsearchAddress: v.GetString("ELASTICSEARCH_ADDRESS"),
	}
	if elasticsearchEnv.ElasticsearchAddress == "" {
		return nil, errors.New("elasticsearch environment variables are empty")
	}

	gomailEnv := GomailEnv{
		MailUsername: v.GetString("MAIL_USERNAME"),
		MailPassword: v.GetString("MAIL_PASSWORD"),
	}
	if gomailEnv.MailUsername == "" {
		return nil, errors.New("redis environment variables are empty")
	}

	redisEnv := RedisEnv{
		RedisAddress:  v.GetString("REDIS_ADDRESS"),
		RedisPassword: v.GetString("REDIS_PASSWORD"),
		RedisDb:       v.GetInt("REDIS_DB"),
	}
	if redisEnv.RedisAddress == "" || redisEnv.RedisDb < 0 {
		return nil, errors.New("redis environment variables are empty")
	}

	loggerEnv := LoggerEnv{
		Level:      v.GetString("ZAP_LEVEL"),
		FilePath:   v.GetString("ZAP_FILEPATH"),
		MaxSize:    v.GetInt("ZAP_MAXSIZE"),
		MaxAge:     v.GetInt("ZAP_MAXAGE"),
		MaxBackups: v.GetInt("ZAP_MAXBACKUPS"),
	}
	if loggerEnv.Level == "" || loggerEnv.FilePath == "" || loggerEnv.MaxSize <= 0 || loggerEnv.MaxAge <= 0 || loggerEnv.MaxBackups <= 0 {
		return nil, errors.New("logger environment variables are empty or invalid")
	}

	return &Env{
		AuthEnv:          authEnv,
		ElasticsearchEnv: elasticsearchEnv,
		GomailEnv:        gomailEnv,
		RedisEnv:         redisEnv,
		LoggerEnv:        loggerEnv,
	}, nil
}
