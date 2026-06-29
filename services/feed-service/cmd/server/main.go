package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/trungquantrannguyen/threadly/db"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	"google.golang.org/grpc"

	sharedcache "github.com/trungquantrannguyen/threadly/pkg/cache"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	feedcache "github.com/trungquantrannguyen/threadly/services/feed-service/internal/cache"
	feedgrpc "github.com/trungquantrannguyen/threadly/services/feed-service/internal/grpc"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/service"
)

func main() {
	cfg := config.Load("feed-service", "50053")
	log := logger.New(cfg.ServiceName, cfg.AppEnv, cfg.LogLevel)

	dtb, err := db.ConnectPostgres(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	sqlDB, err := dtb.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get database instance")
	}
	defer sqlDB.Close()

	log.Info().Msg("Database connected successfully")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal().
			Err(err).
			Str("port", cfg.Port).
			Msg("Failed to listen for feed service grpc server")
	}

	feedRepo := repository.NewFeedRepository(dtb)

	feedService := service.NewFeedService(feedRepo, log)
	redisClient := sharedcache.NewRedisClient(cfg)
	defer redisClient.Close()
	homeFeedCache := feedcache.NewFeedCache(redisClient, 60*time.Second)

	if err := sharedcache.Ping(context.Background(), redisClient); err != nil {
		log.Warn().Err(err).Msg("Redis unavailable, feed cache disabled or degraded")
	}

	grpcServer := grpc.NewServer()

	feedGrpcServer := feedgrpc.NewFeedServiceServer(cfg, log, feedService, homeFeedCache)
	feedpb.RegisterFeedServiceServer(grpcServer, feedGrpcServer)

	log.Info().
		Str("grpc_port", cfg.Port).
		Str("env", cfg.AppEnv).
		Msg("Starting feed service grpc server")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start feed service grpc server")
	}
}
