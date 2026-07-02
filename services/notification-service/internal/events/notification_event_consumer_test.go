package events

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/dto"
)

type fakeEventBus struct {
	consumeCalled bool
	queueName     string
	routingKeys   []string
	handler       messaging.EventHandler
	err           error
}

func (f *fakeEventBus) Consume(
	ctx context.Context,
	queueName string,
	routingKeys []string,
	handler messaging.EventHandler,
) error {
	f.consumeCalled = true
	f.queueName = queueName
	f.routingKeys = routingKeys
	f.handler = handler

	if f.err != nil {
		return f.err
	}

	return nil
}

type fakeNotificationService struct {
	createCalled bool
	received     messaging.Event
	err          error
}

func (f *fakeNotificationService) CreateFromEvent(ctx context.Context, event messaging.Event) error {
	f.createCalled = true
	f.received = event

	if f.err != nil {
		return f.err
	}

	return nil
}

func (f *fakeNotificationService) GetNotifications(ctx context.Context, userID string, limit int) ([]dto.NotificationResponse, error) {
	return nil, nil
}

func (f *fakeNotificationService) GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationService) MarkNotificationRead(ctx context.Context, userID string, notificationID string) (bool, error) {
	return false, nil
}

func (f *fakeNotificationService) MarkAllNotificationsRead(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func newTestNotificationEventConsumer(
	eventBus *fakeEventBus,
	notificationService *fakeNotificationService,
) *NotificationEventConsumer {
	return NewNotificationEventConsumer(
		eventBus,
		notificationService,
		zerolog.Nop(),
	)
}

func TestStartConsumesExpectedNotificationEvents(t *testing.T) {
	eventBus := &fakeEventBus{}
	notificationService := &fakeNotificationService{}

	consumer := newTestNotificationEventConsumer(eventBus, notificationService)

	err := consumer.Start(context.Background())

	require.NoError(t, err)

	assert.True(t, eventBus.consumeCalled)
	assert.NotEmpty(t, eventBus.queueName)
	assert.NotNil(t, eventBus.handler)

	assert.Contains(t, eventBus.routingKeys, messaging.EventUserFollowed)
	assert.Contains(t, eventBus.routingKeys, messaging.EventPostLiked)
	assert.Contains(t, eventBus.routingKeys, messaging.EventReplyCreated)
	assert.Contains(t, eventBus.routingKeys, messaging.EventPostReposted)
}

func TestStartReturnsEventBusError(t *testing.T) {
	expectedErr := errors.New("consume failed")

	eventBus := &fakeEventBus{
		err: expectedErr,
	}
	notificationService := &fakeNotificationService{}

	consumer := newTestNotificationEventConsumer(eventBus, notificationService)

	err := consumer.Start(context.Background())

	require.ErrorIs(t, err, expectedErr)
	assert.True(t, eventBus.consumeCalled)
}

func TestHandleEventCallsNotificationService(t *testing.T) {
	eventBus := &fakeEventBus{}
	notificationService := &fakeNotificationService{}

	consumer := newTestNotificationEventConsumer(eventBus, notificationService)

	event := messaging.Event{
		EventID:      "event-1",
		Type:         messaging.EventUserFollowed,
		ActorID:      "actor-1",
		TargetUserID: "target-user-1",
	}

	err := consumer.handleEvent(context.Background(), event)

	require.NoError(t, err)

	assert.True(t, notificationService.createCalled)
	assert.Equal(t, event, notificationService.received)
}

func TestHandleEventReturnsNotificationServiceError(t *testing.T) {
	expectedErr := errors.New("create notification failed")

	eventBus := &fakeEventBus{}
	notificationService := &fakeNotificationService{
		err: expectedErr,
	}

	consumer := newTestNotificationEventConsumer(eventBus, notificationService)

	event := messaging.Event{
		EventID:      "event-1",
		Type:         messaging.EventUserFollowed,
		ActorID:      "actor-1",
		TargetUserID: "target-user-1",
	}

	err := consumer.handleEvent(context.Background(), event)

	require.ErrorIs(t, err, expectedErr)

	assert.True(t, notificationService.createCalled)
	assert.Equal(t, event, notificationService.received)
}

func TestStartRegisteredHandlerCanProcessEvent(t *testing.T) {
	eventBus := &fakeEventBus{}
	notificationService := &fakeNotificationService{}

	consumer := newTestNotificationEventConsumer(eventBus, notificationService)

	err := consumer.Start(context.Background())
	require.NoError(t, err)
	require.NotNil(t, eventBus.handler)

	event := messaging.Event{
		EventID:      "event-1",
		Type:         messaging.EventUserFollowed,
		ActorID:      "actor-1",
		TargetUserID: "target-user-1",
	}

	err = eventBus.handler(context.Background(), event)

	require.NoError(t, err)

	assert.True(t, notificationService.createCalled)
	assert.Equal(t, event, notificationService.received)
}
