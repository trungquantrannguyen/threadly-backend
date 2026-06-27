package main

import (
	"fmt"
	"net"

	"github.com/trungquantrannguyen/threadly/db"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	contentgrpc "github.com/trungquantrannguyen/threadly/services/content-service/internal/grpc"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/service"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load("content-service", "50052")
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
			Msg("Failed to listen for content service grpc server")
	}

	postRepo := repository.NewPostRepository(dtb)
	contentService := service.NewContentService(postRepo)

	grpcServer := grpc.NewServer()
	contentGrpcServer := contentgrpc.NewContentServiceServer(cfg, log, contentService)
	contentpb.RegisterContentServiceServer(grpcServer, contentGrpcServer)

	log.Info().
		Str("grpc_port", cfg.Port).
		Str("env", cfg.AppEnv).
		Msg("Starting content service grpc server")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start content service grpc server")
	}
}
