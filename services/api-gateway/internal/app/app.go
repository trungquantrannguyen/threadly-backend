package app

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/handlers"
)

func NewRouter(cfg config.Config, log zerolog.Logger) (*gin.Engine, func() error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logging(log))

	healthHandler := handlers.NewHealthHandler(cfg)

	userClient, err := client.NewUserClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create user service grpc client")
	}

	userHandler := handlers.NewUserHandler(userClient, log)

	contentClient, err := client.NewContentClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create content service grpc client")
	}

	contentHandler := handlers.NewContentHandler(contentClient, log)

	feedClient, err := client.NewFeedClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create feed service grpc client")
	}

	feedHandler := handlers.NewFeedHandler(feedClient, log)

	notificationClient, err := client.NewNotificationClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create feed service grpc client")
	}

	notificationHandler := handlers.NewNotificationHandler(notificationClient, log)

	storageClient, err := client.NewStorageClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create feed service grpc client")
	}

	storageHandler := handlers.NewStorageHandler(storageClient, log)

	router.GET("/health", healthHandler.Check)
	api := router.Group("/api")
	{
		api.GET("/health", healthHandler.Check)

		users := api.Group("/users")
		{
			users.GET("/health", userHandler.GetHealth)
		}
		contents := api.Group("/contents")
		{
			contents.GET("/health", contentHandler.GetHealth)
		}
		feeds := api.Group("/feeds")
		{
			feeds.GET("/health", feedHandler.GetHealth)
		}
		notifications := api.Group("/notifications")
		{
			notifications.GET("/health", notificationHandler.GetHealth)
		}
		storages := api.Group("/storages")
		{
			storages.GET("/health", storageHandler.GetHealth)
		}
	}

	cleanup := func() error {
		var cleanupErr error

		if err := userClient.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close user service grpc client")
			cleanupErr = err
		}

		if err := contentClient.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close content service grpc client")
			cleanupErr = err
		}

		if err := feedClient.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close feed service grpc client")
			cleanupErr = err
		}

		if err := notificationClient.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close notification service grpc client")
			cleanupErr = err
		}

		if err := storageClient.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close storage service grpc client")
			cleanupErr = err
		}

		return cleanupErr
	}
	return router, cleanup
}
