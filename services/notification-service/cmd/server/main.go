package main

import (
	"context"
	"fmt"
	"net"

	"github.com/trungquantrannguyen/threadly/db"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/logger"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	notificationevents "github.com/trungquantrannguyen/threadly/services/notification-service/internal/events"
	notificationgrpc "github.com/trungquantrannguyen/threadly/services/notification-service/internal/grpc"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/service"
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
	notificationRepo := repository.NewNotificationRepository(dtb)
	notificationService := service.NewNotificationService(notificationRepo)

	grpcServer := grpc.NewServer()
	notificationGrpcServer := notificationgrpc.NewNotificationServiceServer(cfg, log, notificationService)
	notificationpb.RegisterNotificationServiceServer(grpcServer, notificationGrpcServer)

	eventBus, err := messaging.NewRabbitMQ(cfg, log)
	if err != nil {
		log.Warn().Err(err).Msg("RabbitMQ unavailable, notification consumer disabled")
	} else {
		defer eventBus.Close()

		notificationConsumer := notificationevents.NewNotificationEventConsumer(
			eventBus,
			notificationService,
			log,
		)

		if err := notificationConsumer.Start(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Failed to start notification event consumer")
		} else {
			log.Info().Msg("Notification event consumer started")
		}
	}

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
