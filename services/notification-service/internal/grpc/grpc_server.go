package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
)

type NotificationServiceServer struct {
	notificationpb.UnimplementedNotificationServiceServer
	cfg config.Config
	log zerolog.Logger
}

func NewNotificationServiceServer(cfg config.Config, log zerolog.Logger) *NotificationServiceServer {
	return &NotificationServiceServer{
		cfg: cfg,
		log: log,
	}
}

func (s *NotificationServiceServer) GetHealth(ctx context.Context, req *notificationpb.GetNotificationServiceHealthRequest) (*notificationpb.GetNotificationServiceHealthResponse, error) {
	s.log.Info().Msg("Notification service is healthy")
	return &notificationpb.GetNotificationServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}
