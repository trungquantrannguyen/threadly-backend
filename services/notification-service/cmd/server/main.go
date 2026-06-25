package main

import (
	"fmt"
	"net"

	"github.com/trungquantrannguyen/threadly/db"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	notificationgrpc "github.com/trungquantrannguyen/threadly/services/notification-service/internal/grpc"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load("notification-service", "50054")
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
			Msg("Failed to listen for notification service grpc server")
	}

	grpcServer := grpc.NewServer()
	notificationGrpcServer := notificationgrpc.NewNotificationServiceServer(cfg, log)
	notificationpb.RegisterNotificationServiceServer(grpcServer, notificationGrpcServer)

	log.Info().
		Str("grpc_port", cfg.Port).
		Str("env", cfg.AppEnv).
		Msg("Starting notification service grpc server")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start notification service grpc server")
	}
}
