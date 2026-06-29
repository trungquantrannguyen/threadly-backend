package events

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/cache"
)

const feedCacheInvalidationQueue = "feed-service.cache-invalidation"

type FeedEventConsumer struct {
	eventBus  *messaging.RabbitMQ
	feedCache cache.FeedCache
	log       zerolog.Logger
}

func NewFeedEventConsumer(
	eventBus *messaging.RabbitMQ,
	feedCache cache.FeedCache,
	log zerolog.Logger,
) *FeedEventConsumer {
	return &FeedEventConsumer{
		eventBus:  eventBus,
		feedCache: feedCache,
		log:       log,
	}
}

func (c *FeedEventConsumer) Start(ctx context.Context) error {
	return c.eventBus.Consume(
		ctx,
		feedCacheInvalidationQueue,
		[]string{
			messaging.EventPostCreated,
			messaging.EventUserFollowed,
		},
		c.handleEvent,
	)
}

func (c *FeedEventConsumer) handleEvent(ctx context.Context, event messaging.Event) error {
	switch event.Type {
	case messaging.EventPostCreated:
		return c.clearUserHomeFeed(ctx, event.AuthorID, event.Type)

	case messaging.EventUserFollowed:
		return c.clearUserHomeFeed(ctx, event.ActorID, event.Type)

	default:
		return nil
	}
}

func (c *FeedEventConsumer) clearUserHomeFeed(ctx context.Context, userID string, eventType string) error {
	if userID == "" {
		return nil
	}

	if err := c.feedCache.DeleteHomeFeedByUserID(ctx, userID); err != nil {
		return err
	}

	c.log.Info().
		Str("user_id", userID).
		Str("event_type", eventType).
		Msg("cleared home feed cache")

	return nil
}
