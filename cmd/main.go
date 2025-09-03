package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
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
	"go.uber.org/zap"
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
	defer redisRawClient.Close()
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
	reportWorker.Start()
	defer reportWorker.Stop()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://report.localhost", "http://swagger.localhost"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	reportHandler.SetupRoutes(r)
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":8084",
		Handler: r,
	}

	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown failed", zap.Error(err))
		}
		logger.Info("Report service stopped gracefully")
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to run service: %v", err)
	}
}
