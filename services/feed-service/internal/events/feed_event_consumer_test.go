package events

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
)

type fakeFeedCache struct {
	deletedUserIDs []string
	err            error
}

type fakeEventBus struct {
	queueName   string
	routingKeys []string
	handler     messaging.EventHandler
	err         error
}

func (f *fakeEventBus) Consume(
	ctx context.Context,
	queueName string,
	routingKeys []string,
	handler messaging.EventHandler,
) error {
	f.queueName = queueName
	f.routingKeys = routingKeys
	f.handler = handler

	if f.err != nil {
		return f.err
	}

	return nil
}

func (f *fakeFeedCache) GetHomeFeed(ctx context.Context, userID string, limit int32, cursor string) (*feedpb.HomeFeedResponse, error) {
	return nil, nil
}

func (f *fakeFeedCache) SetHomeFeed(ctx context.Context, userID string, limit int32, cursor string, feed *feedpb.HomeFeedResponse) error {
	return nil
}

func (f *fakeFeedCache) DeleteHomeFeedByUserID(ctx context.Context, userID string) error {
	f.deletedUserIDs = append(f.deletedUserIDs, userID)

	if f.err != nil {
		return f.err
	}

	return nil
}

func newTestFeedEventConsumer(feedCache *fakeFeedCache) *FeedEventConsumer {
	return NewFeedEventConsumer(
		nil,
		feedCache,
		zerolog.Nop(),
	)
}

func TestHandleEventPostCreatedClearsAuthorFeed(t *testing.T) {
	feedCache := &fakeFeedCache{}
	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.handleEvent(context.Background(), messaging.Event{
		Type:     messaging.EventPostCreated,
		AuthorID: "author-1",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"author-1"}, feedCache.deletedUserIDs)
}

func TestHandleEventPostDeletedClearsAuthorFeed(t *testing.T) {
	feedCache := &fakeFeedCache{}
	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.handleEvent(context.Background(), messaging.Event{
		Type:     messaging.EventPostDeleted,
		AuthorID: "author-1",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"author-1"}, feedCache.deletedUserIDs)
}

func TestHandleEventUserFollowedClearsActorFeed(t *testing.T) {
	feedCache := &fakeFeedCache{}
	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.handleEvent(context.Background(), messaging.Event{
		Type:    messaging.EventUserFollowed,
		ActorID: "actor-1",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"actor-1"}, feedCache.deletedUserIDs)
}

func TestHandleEventUserUnfollowedClearsActorFeed(t *testing.T) {
	feedCache := &fakeFeedCache{}
	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.handleEvent(context.Background(), messaging.Event{
		Type:    messaging.EventUserUnfollowed,
		ActorID: "actor-1",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"actor-1"}, feedCache.deletedUserIDs)
}

func TestHandleEventUnknownEventDoesNothing(t *testing.T) {
	feedCache := &fakeFeedCache{}
	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.handleEvent(context.Background(), messaging.Event{
		Type:     "unknown.event",
		AuthorID: "author-1",
		ActorID:  "actor-1",
	})

	require.NoError(t, err)
	assert.Empty(t, feedCache.deletedUserIDs)
}

func TestStartConsumesExpectedEvents(t *testing.T) {
	eventBus := &fakeEventBus{}
	feedCache := &fakeFeedCache{}

	consumer := NewFeedEventConsumer(
		eventBus,
		feedCache,
		zerolog.Nop(),
	)

	err := consumer.Start(context.Background())

	require.NoError(t, err)

	assert.Equal(t, feedCacheInvalidationQueue, eventBus.queueName)

	assert.ElementsMatch(t, []string{
		messaging.EventPostCreated,
		messaging.EventPostDeleted,
		messaging.EventUserFollowed,
		messaging.EventUserUnfollowed,
	}, eventBus.routingKeys)

	require.NotNil(t, eventBus.handler)
}

func TestStartReturnsEventBusError(t *testing.T) {
	expectedErr := errors.New("rabbitmq consume failed")

	eventBus := &fakeEventBus{
		err: expectedErr,
	}

	consumer := NewFeedEventConsumer(
		eventBus,
		&fakeFeedCache{},
		zerolog.Nop(),
	)

	err := consumer.Start(context.Background())

	require.ErrorIs(t, err, expectedErr)
}

func TestClearUserHomeFeedDoesNothingWhenUserIDEmpty(t *testing.T) {
	feedCache := &fakeFeedCache{}
	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.clearUserHomeFeed(context.Background(), "", messaging.EventPostCreated)

	require.NoError(t, err)
	assert.Empty(t, feedCache.deletedUserIDs)
}

func TestClearUserHomeFeedReturnsCacheError(t *testing.T) {
	expectedErr := errors.New("redis delete failed")

	feedCache := &fakeFeedCache{
		err: expectedErr,
	}

	consumer := newTestFeedEventConsumer(feedCache)

	err := consumer.clearUserHomeFeed(context.Background(), "user-1", messaging.EventPostCreated)

	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, []string{"user-1"}, feedCache.deletedUserIDs)
}
