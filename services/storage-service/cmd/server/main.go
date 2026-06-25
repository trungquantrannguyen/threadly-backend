package main

import (
	"fmt"
	"net"

	"github.com/trungquantrannguyen/threadly/db"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	storagepb "github.com/trungquantrannguyen/threadly/proto/storage"
	storagegrpc "github.com/trungquantrannguyen/threadly/services/storage-service/internal/grpc"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load("storage-service", "50055")
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
			Msg("Failed to listen for storage service grpc server")
	}

	grpcServer := grpc.NewServer()
	storageGrpcServer := storagegrpc.NewStorageServiceServer(cfg, log)
	storagepb.RegisterStorageServiceServer(grpcServer, storageGrpcServer)

	log.Info().
		Str("grpc_port", cfg.Port).
		Str("env", cfg.AppEnv).
		Msg("Starting storage service grpc server")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start storage service grpc server")
	}
}
