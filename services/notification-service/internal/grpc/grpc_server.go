package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/service"
)

type NotificationServiceServer struct {
	notificationpb.UnimplementedNotificationServiceServer
	cfg                 config.Config
	log                 zerolog.Logger
	notificationService service.NotificationService
}

func NewNotificationServiceServer(cfg config.Config, log zerolog.Logger, notificationService service.NotificationService) *NotificationServiceServer {
	return &NotificationServiceServer{
		cfg:                 cfg,
		log:                 log,
		notificationService: notificationService,
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

func (s *NotificationServiceServer) GetNotifications(ctx context.Context, req *notificationpb.GetNotificationsRequest) (*notificationpb.GetNotificationsResponse, error) {
	notifications, err := s.notificationService.GetNotifications(ctx, req.GetUserId(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}

	res := make([]*notificationpb.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		res = append(res, &notificationpb.NotificationResponse{
			Id:          n.ID,
			RecipientId: n.RecipientID,
			ActorId:     n.ActorID,
			Type:        n.Type,
			EntityType:  n.EntityType,
			EntityId:    n.EntityID,
			Payload:     n.Payload,
			ReadAt:      n.ReadAt,
			CreatedAt:   n.CreatedAt,
		})
	}

	return &notificationpb.GetNotificationsResponse{
		Notifications: res,
	}, nil
}

func (s *NotificationServiceServer) MarkNotificationRead(ctx context.Context, req *notificationpb.MarkNotificationReadRequest) (*notificationpb.MarkNotificationReadResponse, error) {
	updated, err := s.notificationService.MarkNotificationRead(ctx, req.GetUserId(), req.GetNotificationId())
	if err != nil {
		return nil, err
	}

	message := "notification marked as read"
	if !updated {
		message = "notification not found or already read"
	}

	return &notificationpb.MarkNotificationReadResponse{
		Success: updated,
		Message: message,
	}, nil
}

func (s *NotificationServiceServer) MarkAllNotificationsRead(ctx context.Context, req *notificationpb.MarkAllNotificationsReadRequest) (*notificationpb.MarkAllNotificationsReadResponse, error) {
	count, err := s.notificationService.MarkAllNotificationsRead(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	return &notificationpb.MarkAllNotificationsReadResponse{
		Success:      true,
		Message:      "notifications marked as read",
		UpdatedCount: int32(count),
	}, nil
}
