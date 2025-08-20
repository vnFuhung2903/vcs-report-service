package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"github.com/vnFuhung2903/vcs-report-service/api"
	_ "github.com/vnFuhung2903/vcs-report-service/docs"
	"github.com/vnFuhung2903/vcs-report-service/infrastructures/databases"
	"github.com/vnFuhung2903/vcs-report-service/interfaces"
	"github.com/vnFuhung2903/vcs-report-service/pkg/env"
	"github.com/vnFuhung2903/vcs-report-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-report-service/pkg/middlewares"
	"github.com/vnFuhung2903/vcs-report-service/usecases/services"
	"github.com/vnFuhung2903/vcs-report-service/usecases/workers"
)

// @title VCS SMS API
// @version 1.0
// @description Container Management System API
// @host localhost:8084
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	env, err := env.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to retrieve env: %v", err)
	}

	logger, err := logger.LoadLogger(env.LoggerEnv)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}

	esRawClient, err := databases.NewElasticsearchFactory(env.ElasticsearchEnv).ConnectElasticsearch()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}
	esClient := interfaces.NewElasticsearchClient(esRawClient)

	redisRawClient := databases.NewRedisFactory(env.RedisEnv).ConnectRedis()
	redisClient := interfaces.NewRedisClient(redisRawClient)

	jwtMiddleware := middlewares.NewJWTMiddleware(env.AuthEnv)

	reportService := services.NewReportService(esClient, redisClient, logger, env.GomailEnv)
	reportHandler := api.NewReportHandler(reportService, jwtMiddleware)

	reportWorker := workers.NewReportkWorker(
		reportService,
		"hung29032004@gmail.com",
		logger,
		24*time.Hour,
	)
	reportWorker.Start(1)

	r := gin.Default()
	reportHandler.SetupRoutes(r)
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("Shutting down...")
		reportWorker.Stop()
		os.Exit(0)
	}()
	if err := r.Run(":8084"); err != nil {
		log.Fatalf("Failed to run service: %v", err)
	} else {
		logger.Info("Report service is running on port 8084")
	}
}
