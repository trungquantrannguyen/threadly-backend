package app

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/trungquantrannguyen/threadly/pkg/cache"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/handlers"
)

func NewRouter(cfg config.Config, log zerolog.Logger) (*gin.Engine, func() error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	redisClient := cache.NewRedisClient(cfg)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logging(log))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	healthHandler := handlers.NewHealthHandler(cfg)

	userClient, err := client.NewUserClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create user service grpc client")
	}

	contentClient, err := client.NewContentClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create content service grpc client")
	}

	feedClient, err := client.NewFeedClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create feed service grpc client")
	}

	notificationClient, err := client.NewNotificationClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create notification service grpc client")
	}

	storageClient, err := client.NewStorageClient(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create storage service grpc client")
	}

	userHandler := handlers.NewUserHandler(userClient, log, contentClient)
	contentHandler := handlers.NewContentHandler(contentClient, log)
	feedHandler := handlers.NewFeedHandler(feedClient, log)
	notificationHandler := handlers.NewNotificationHandler(notificationClient, log)
	storageHandler := handlers.NewStorageHandler(storageClient, log)

	router.GET("/health", healthHandler.Check)
	api := router.Group("/api")
	api.Use(middleware.RedisRateLimiter(redisClient, 100, time.Minute))
	{
		api.GET("/health", healthHandler.Check)

		users := api.Group("/users")
		{
			users.GET("/health", userHandler.GetHealth)
			users.POST("/register", middleware.RedisRateLimiter(redisClient, 10, time.Minute), userHandler.Register)
			users.POST("/login", middleware.RedisRateLimiter(redisClient, 10, time.Minute), userHandler.Login)
			users.POST("/refresh", middleware.RedisRateLimiter(redisClient, 30, time.Minute), userHandler.RefreshToken)
			protectedUsers := users.Group("")
			protectedUsers.Use(middleware.AuthMiddleware(cfg))
			{
				protectedUsers.POST("/logout", userHandler.Logout)
				protectedUsers.GET("/me", userHandler.GetMe)
				protectedUsers.PATCH("/me", userHandler.UpdateProfile)
				protectedUsers.DELETE("/me", userHandler.DeleteUser)
			}
		}
		contents := api.Group("/contents")
		{
			contents.GET("/health", contentHandler.GetHealth)
			protectedContent := contents.Group("/posts")
			protectedContent.Use(middleware.AuthMiddleware(cfg))
			{
				protectedContent.POST("", contentHandler.CreatePost)
				protectedPost := protectedContent.Group("/:postID")
				{

					protectedPost.GET("", contentHandler.GetPost)
					protectedPost.DELETE("", contentHandler.DeletePost)
					protectedPostReply := protectedPost.Group("/replies")
					{
						protectedPostReply.POST("", contentHandler.CreateReply)
						protectedPostReply.GET("", contentHandler.GetReplies)
					}
					protectedPost.POST("/likes", contentHandler.LikePost)
					protectedPost.DELETE("/likes", contentHandler.UnlikePost)

					protectedPost.POST("/bookmarks", contentHandler.BookmarkPost)
					protectedPost.DELETE("/bookmarks", contentHandler.UnbookmarkPost)

					protectedPost.POST("/reposts", contentHandler.RepostPost)
					protectedPost.DELETE("/reposts", contentHandler.UndoRepost)
				}
			}
			protectedContentUsers := contents.Group("/users")
			protectedContentUsers.Use(middleware.AuthMiddleware(cfg))
			{
				protectedContentUsers.POST("/:userID/follow", contentHandler.FollowUser)
				protectedContentUsers.DELETE("/:userID/follow", contentHandler.UnfollowUser)
				protectedContentUsers.GET("/:userID/followers", contentHandler.GetFollowers)
				protectedContentUsers.GET("/:userID/following", contentHandler.GetFollowing)
			}
		}
		feeds := api.Group("/feeds")
		{
			feeds.GET("/health", feedHandler.GetHealth)
			protectedFeeds := feeds.Group("")
			protectedFeeds.Use(middleware.AuthMiddleware(cfg))
			{
				protectedFeeds.GET("/home", feedHandler.GetHomeFeed)
			}
		}
		notifications := api.Group("/notifications")
		{
			notifications.GET("/health", notificationHandler.GetHealth)
			protectedNotifications := notifications.Group("")
			protectedNotifications.Use(middleware.AuthMiddleware(cfg))
			{
				protectedNotifications.GET("", notificationHandler.GetNotifications)
				protectedNotifications.GET("/unread-count", notificationHandler.GetUnreadNotificationCount)
				protectedNotifications.PATCH("/:id/read", notificationHandler.MarkNotificationRead)
				protectedNotifications.PATCH("/read-all", notificationHandler.MarkAllNotificationsRead)
			}
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
