package main

import (
	"fmt"
	"net"

	"github.com/trungquantrannguyen/threadly/db"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	usergrpc "github.com/trungquantrannguyen/threadly/services/user-service/internal/grpc"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load("user-service", "50051")
	log := logger.New(cfg.ServiceName, cfg.AppEnv)
	dtb, err := db.ConntectPostgres(cfg)
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
			Msg("Failed to listen for user service grpc server")
	}

	grpcServer := grpc.NewServer()
	userGrpcServer := usergrpc.NewUserServiceServer(cfg, log)
	userpb.RegisterUserServiceServer(grpcServer, userGrpcServer)

	log.Info().
		Str("grpc_port", cfg.Port).
		Str("env", cfg.AppEnv).
		Msg("Starting user service grpc server")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start user service grpc server")
	}
}
