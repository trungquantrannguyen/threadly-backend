package events

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/service"
)

const notificationQueue = "notification-service.events"

type NotificationEventConsumer struct {
	eventBus            *messaging.RabbitMQ
	notificationService service.NotificationService
	log                 zerolog.Logger
}

func NewNotificationEventConsumer(
	eventBus *messaging.RabbitMQ,
	notificationService service.NotificationService,
	log zerolog.Logger,
) *NotificationEventConsumer {
	return &NotificationEventConsumer{
		eventBus:            eventBus,
		notificationService: notificationService,
		log:                 log,
	}
}

func (c *NotificationEventConsumer) Start(ctx context.Context) error {
	return c.eventBus.Consume(
		ctx,
		notificationQueue,
		[]string{
			messaging.EventUserFollowed,
			messaging.EventPostLiked,
			messaging.EventReplyCreated,
			messaging.EventPostReposted,
		},
		c.handleEvent,
	)
}

func (c *NotificationEventConsumer) handleEvent(ctx context.Context, event messaging.Event) error {
	if err := c.notificationService.CreateFromEvent(ctx, event); err != nil {
		return err
	}

	c.log.Info().
		Str("event_type", event.Type).
		Str("actor_id", event.ActorID).
		Str("target_user_id", event.TargetUserID).
		Msg("notification event processed")

	return nil
}
