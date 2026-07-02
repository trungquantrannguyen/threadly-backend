package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"gorm.io/datatypes"
)

type fakeNotificationRepo struct {
	createdNotifications []*dbmodel.Notification
	createErr            error

	findResult      []dbmodel.Notification
	findErr         error
	findRecipientID uuid.UUID
	findLimit       int

	countResult      int64
	countErr         error
	countRecipientID uuid.UUID

	markAsReadResult      bool
	markAsReadErr         error
	markAsReadID          uuid.UUID
	markAsReadRecipientID uuid.UUID

	markAllResult      int64
	markAllErr         error
	markAllRecipientID uuid.UUID
}

func (f *fakeNotificationRepo) Create(ctx context.Context, notification *dbmodel.Notification) error {
	if f.createErr != nil {
		return f.createErr
	}

	copied := *notification
	if copied.ID == uuid.Nil {
		copied.ID = uuid.New()
	}
	if copied.CreatedAt.IsZero() {
		copied.CreatedAt = time.Now().UTC()
	}

	f.createdNotifications = append(f.createdNotifications, &copied)

	return nil
}

func (f *fakeNotificationRepo) FindByRecipientID(ctx context.Context, recipientID uuid.UUID, limit int) ([]dbmodel.Notification, error) {
	f.findRecipientID = recipientID
	f.findLimit = limit

	if f.findErr != nil {
		return nil, f.findErr
	}

	return f.findResult, nil
}

func (f *fakeNotificationRepo) CountUnreadByRecipientID(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	f.countRecipientID = recipientID

	if f.countErr != nil {
		return 0, f.countErr
	}

	return f.countResult, nil
}

func (f *fakeNotificationRepo) MarkAsRead(ctx context.Context, notificationID uuid.UUID, recipientID uuid.UUID) (bool, error) {
	f.markAsReadID = notificationID
	f.markAsReadRecipientID = recipientID

	if f.markAsReadErr != nil {
		return false, f.markAsReadErr
	}

	return f.markAsReadResult, nil
}

func (f *fakeNotificationRepo) MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	f.markAllRecipientID = recipientID

	if f.markAllErr != nil {
		return 0, f.markAllErr
	}

	return f.markAllResult, nil
}

func newTestNotificationService(repo *fakeNotificationRepo) NotificationService {
	return NewNotificationService(repo)
}

func TestCreateFromEventUserFollowedSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	eventID := uuid.New()
	actorID := uuid.New()
	targetUserID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      eventID.String(),
		Type:         messaging.EventUserFollowed,
		ActorID:      actorID.String(),
		TargetUserID: targetUserID.String(),
	})

	require.NoError(t, err)
	require.Len(t, repo.createdNotifications, 1)

	notification := repo.createdNotifications[0]

	assert.Equal(t, eventID, notification.EventID)
	assert.Equal(t, targetUserID, notification.RecipientID)
	require.NotNil(t, notification.ActorID)
	assert.Equal(t, actorID, *notification.ActorID)
	assert.Equal(t, NotificationTypeUserFollowed, notification.Type)
	assert.Equal(t, EntityTypeUser, notification.EntityType)
	require.NotNil(t, notification.EntityID)
	assert.Equal(t, actorID, *notification.EntityID)
	assert.JSONEq(t, `{"message":"started following you"}`, string(notification.Payload))
}

func TestCreateFromEventUserFollowedSelfDoesNothing(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	userID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventUserFollowed,
		ActorID:      userID.String(),
		TargetUserID: userID.String(),
	})

	require.NoError(t, err)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventUserFollowedMissingFields(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID: uuid.NewString(),
		Type:    messaging.EventUserFollowed,
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventUserFollowedInvalidActorID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventUserFollowed,
		ActorID:      "bad-actor-id",
		TargetUserID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventUserFollowedInvalidTargetUserID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventUserFollowed,
		ActorID:      uuid.NewString(),
		TargetUserID: "bad-target-id",
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventPostLikedSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	eventID := uuid.New()
	actorID := uuid.New()
	authorID := uuid.New()
	postID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:  eventID.String(),
		Type:     messaging.EventPostLiked,
		ActorID:  actorID.String(),
		AuthorID: authorID.String(),
		PostID:   postID.String(),
	})

	require.NoError(t, err)
	require.Len(t, repo.createdNotifications, 1)

	notification := repo.createdNotifications[0]

	assert.Equal(t, eventID, notification.EventID)
	assert.Equal(t, authorID, notification.RecipientID)
	require.NotNil(t, notification.ActorID)
	assert.Equal(t, actorID, *notification.ActorID)
	assert.Equal(t, NotificationTypePostLiked, notification.Type)
	assert.Equal(t, EntityTypePost, notification.EntityType)
	require.NotNil(t, notification.EntityID)
	assert.Equal(t, postID, *notification.EntityID)
	assert.JSONEq(t, `{"message":"liked your post"}`, string(notification.Payload))
}

func TestCreateFromEventPostRepostedSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	actorID := uuid.New()
	authorID := uuid.New()
	postID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:  uuid.NewString(),
		Type:     messaging.EventPostReposted,
		ActorID:  actorID.String(),
		AuthorID: authorID.String(),
		PostID:   postID.String(),
	})

	require.NoError(t, err)
	require.Len(t, repo.createdNotifications, 1)

	notification := repo.createdNotifications[0]

	assert.Equal(t, NotificationTypePostReposted, notification.Type)
	assert.Equal(t, EntityTypePost, notification.EntityType)
	assert.JSONEq(t, `{"message":"reposted your post"}`, string(notification.Payload))
}

func TestCreateFromEventPostNotificationSelfDoesNothing(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	userID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:  uuid.NewString(),
		Type:     messaging.EventPostLiked,
		ActorID:  userID.String(),
		AuthorID: userID.String(),
		PostID:   uuid.NewString(),
	})

	require.NoError(t, err)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventPostNotificationMissingFields(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID: uuid.NewString(),
		Type:    messaging.EventPostLiked,
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventPostNotificationInvalidActorID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:  uuid.NewString(),
		Type:     messaging.EventPostLiked,
		ActorID:  "bad-actor-id",
		AuthorID: uuid.NewString(),
		PostID:   uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventPostNotificationInvalidAuthorID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:  uuid.NewString(),
		Type:     messaging.EventPostLiked,
		ActorID:  uuid.NewString(),
		AuthorID: "bad-author-id",
		PostID:   uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventPostNotificationInvalidPostID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:  uuid.NewString(),
		Type:     messaging.EventPostLiked,
		ActorID:  uuid.NewString(),
		AuthorID: uuid.NewString(),
		PostID:   "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventReplyCreatedSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	eventID := uuid.New()
	actorID := uuid.New()
	targetUserID := uuid.New()
	replyID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      eventID.String(),
		Type:         messaging.EventReplyCreated,
		ActorID:      actorID.String(),
		TargetUserID: targetUserID.String(),
		PostID:       replyID.String(),
	})

	require.NoError(t, err)
	require.Len(t, repo.createdNotifications, 1)

	notification := repo.createdNotifications[0]

	assert.Equal(t, eventID, notification.EventID)
	assert.Equal(t, targetUserID, notification.RecipientID)
	require.NotNil(t, notification.ActorID)
	assert.Equal(t, actorID, *notification.ActorID)
	assert.Equal(t, NotificationTypePostReplied, notification.Type)
	assert.Equal(t, EntityTypePost, notification.EntityType)
	require.NotNil(t, notification.EntityID)
	assert.Equal(t, replyID, *notification.EntityID)
	assert.JSONEq(t, `{"message":"replied to your post"}`, string(notification.Payload))
}

func TestCreateFromEventReplyCreatedSelfDoesNothing(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	userID := uuid.New()

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventReplyCreated,
		ActorID:      userID.String(),
		TargetUserID: userID.String(),
		PostID:       uuid.NewString(),
	})

	require.NoError(t, err)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventReplyCreatedMissingFields(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID: uuid.NewString(),
		Type:    messaging.EventReplyCreated,
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventReplyCreatedInvalidActorID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventReplyCreated,
		ActorID:      "bad-actor-id",
		TargetUserID: uuid.NewString(),
		PostID:       uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventReplyCreatedInvalidTargetUserID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventReplyCreated,
		ActorID:      uuid.NewString(),
		TargetUserID: "bad-target-id",
		PostID:       uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventReplyCreatedInvalidPostID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventReplyCreated,
		ActorID:      uuid.NewString(),
		TargetUserID: uuid.NewString(),
		PostID:       "bad-post-id",
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventInvalidEventID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      "bad-event-id",
		Type:         messaging.EventUserFollowed,
		ActorID:      uuid.NewString(),
		TargetUserID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventMissingEventID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		Type:         messaging.EventUserFollowed,
		ActorID:      uuid.NewString(),
		TargetUserID: uuid.NewString(),
	})

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventUnknownTypeDoesNothing(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID: uuid.NewString(),
		Type:    "unknown.event",
	})

	require.NoError(t, err)
	assert.Empty(t, repo.createdNotifications)
}

func TestCreateFromEventRepositoryError(t *testing.T) {
	expectedErr := errors.New("create notification failed")
	repo := &fakeNotificationRepo{
		createErr: expectedErr,
	}
	svc := newTestNotificationService(repo)

	err := svc.CreateFromEvent(context.Background(), messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventUserFollowed,
		ActorID:      uuid.NewString(),
		TargetUserID: uuid.NewString(),
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestGetUnreadNotificationCountSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{
		countResult: 5,
	}
	svc := newTestNotificationService(repo)

	userID := uuid.New()

	count, err := svc.GetUnreadNotificationCount(context.Background(), userID.String())

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.Equal(t, userID, repo.countRecipientID)
}

func TestGetUnreadNotificationCountInvalidUserID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	count, err := svc.GetUnreadNotificationCount(context.Background(), "bad-user-id")

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Equal(t, int64(0), count)
}

func TestGetUnreadNotificationCountRepositoryError(t *testing.T) {
	expectedErr := errors.New("count failed")
	repo := &fakeNotificationRepo{
		countErr: expectedErr,
	}
	svc := newTestNotificationService(repo)

	count, err := svc.GetUnreadNotificationCount(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, int64(0), count)
}

func TestGetNotificationsSuccess(t *testing.T) {
	avatarURL := "https://example.com/avatar.png"
	readAt := time.Now().UTC()
	createdAt := time.Now().UTC().Add(-time.Hour)

	recipientID := uuid.New()
	actorID := uuid.New()
	entityID := uuid.New()
	notificationID := uuid.New()

	repo := &fakeNotificationRepo{
		findResult: []dbmodel.Notification{
			{
				ID:          notificationID,
				EventID:     uuid.New(),
				RecipientID: recipientID,
				ActorID:     &actorID,
				Type:        NotificationTypePostLiked,
				EntityType:  EntityTypePost,
				EntityID:    &entityID,
				Payload:     datatypes.JSON([]byte(`{"message":"liked your post"}`)),
				ReadAt:      &readAt,
				CreatedAt:   createdAt,
				Actor: &dbmodel.User{
					ID:          actorID,
					Username:    "actor",
					DisplayName: "Actor User",
					AvatarURL:   &avatarURL,
					IsVerified:  true,
				},
			},
		},
	}
	svc := newTestNotificationService(repo)

	res, err := svc.GetNotifications(context.Background(), recipientID.String(), 10)

	require.NoError(t, err)
	require.Len(t, res, 1)

	assert.Equal(t, recipientID, repo.findRecipientID)
	assert.Equal(t, 10, repo.findLimit)

	assert.Equal(t, notificationID.String(), res[0].ID)
	assert.Equal(t, recipientID.String(), res[0].RecipientID)
	assert.Equal(t, actorID.String(), res[0].ActorID)
	assert.Equal(t, NotificationTypePostLiked, res[0].Type)
	assert.Equal(t, EntityTypePost, res[0].EntityType)
	assert.Equal(t, entityID.String(), res[0].EntityID)
	assert.JSONEq(t, `{"message":"liked your post"}`, res[0].Payload)
	assert.Equal(t, readAt.Format(time.RFC3339), res[0].ReadAt)
	assert.Equal(t, createdAt.Format(time.RFC3339), res[0].CreatedAt)

	require.NotNil(t, res[0].Actor)
	assert.Equal(t, actorID.String(), res[0].Actor.ID)
	assert.Equal(t, "actor", res[0].Actor.Username)
	assert.Equal(t, "Actor User", res[0].Actor.DisplayName)
	assert.Equal(t, avatarURL, res[0].Actor.AvatarURL)
	assert.True(t, res[0].Actor.IsVerified)
}

func TestGetNotificationsDefaultsLimitWhenInvalid(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	_, err := svc.GetNotifications(context.Background(), uuid.NewString(), 0)

	require.NoError(t, err)
	assert.Equal(t, 20, repo.findLimit)
}

func TestGetNotificationsDefaultsLimitWhenTooLarge(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	_, err := svc.GetNotifications(context.Background(), uuid.NewString(), 100)

	require.NoError(t, err)
	assert.Equal(t, 20, repo.findLimit)
}

func TestGetNotificationsInvalidUserID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	res, err := svc.GetNotifications(context.Background(), "bad-user-id", 10)

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Nil(t, res)
}

func TestGetNotificationsRepositoryError(t *testing.T) {
	expectedErr := errors.New("find notifications failed")
	repo := &fakeNotificationRepo{
		findErr: expectedErr,
	}
	svc := newTestNotificationService(repo)

	res, err := svc.GetNotifications(context.Background(), uuid.NewString(), 10)

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestNotificationResponseHandlesNilActorEntityAndReadAt(t *testing.T) {
	recipientID := uuid.New()
	createdAt := time.Now().UTC()

	notification := dbmodel.Notification{
		ID:          uuid.New(),
		EventID:     uuid.New(),
		RecipientID: recipientID,
		Type:        NotificationTypeUserFollowed,
		EntityType:  EntityTypeUser,
		Payload:     datatypes.JSON([]byte(`{"message":"started following you"}`)),
		CreatedAt:   createdAt,
	}

	res := toNotificationResponse(notification)

	assert.Equal(t, "", res.ActorID)
	assert.Equal(t, "", res.EntityID)
	assert.Equal(t, "", res.ReadAt)
	assert.Nil(t, res.Actor)
	assert.Equal(t, createdAt.Format(time.RFC3339), res.CreatedAt)
}

func TestNotificationResponseActorWithoutAvatar(t *testing.T) {
	actorID := uuid.New()
	recipientID := uuid.New()
	createdAt := time.Now().UTC()

	notification := dbmodel.Notification{
		ID:          uuid.New(),
		EventID:     uuid.New(),
		RecipientID: recipientID,
		ActorID:     &actorID,
		Type:        NotificationTypeUserFollowed,
		EntityType:  EntityTypeUser,
		Payload:     datatypes.JSON([]byte(`{"message":"started following you"}`)),
		CreatedAt:   createdAt,
		Actor: &dbmodel.User{
			ID:          actorID,
			Username:    "noavatar",
			DisplayName: "No Avatar",
			IsVerified:  false,
		},
	}

	res := toNotificationResponse(notification)

	require.NotNil(t, res.Actor)
	assert.Equal(t, actorID.String(), res.Actor.ID)
	assert.Equal(t, "noavatar", res.Actor.Username)
	assert.Equal(t, "No Avatar", res.Actor.DisplayName)
	assert.Equal(t, "", res.Actor.AvatarURL)
	assert.False(t, res.Actor.IsVerified)
}

func TestMarkNotificationReadSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{
		markAsReadResult: true,
	}
	svc := newTestNotificationService(repo)

	userID := uuid.New()
	notificationID := uuid.New()

	updated, err := svc.MarkNotificationRead(context.Background(), userID.String(), notificationID.String())

	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, userID, repo.markAsReadRecipientID)
	assert.Equal(t, notificationID, repo.markAsReadID)
}

func TestMarkNotificationReadInvalidUserID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	updated, err := svc.MarkNotificationRead(context.Background(), "bad-user-id", uuid.NewString())

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.False(t, updated)
}

func TestMarkNotificationReadInvalidNotificationID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	updated, err := svc.MarkNotificationRead(context.Background(), uuid.NewString(), "bad-notification-id")

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.False(t, updated)
}

func TestMarkNotificationReadRepositoryError(t *testing.T) {
	expectedErr := errors.New("mark read failed")
	repo := &fakeNotificationRepo{
		markAsReadErr: expectedErr,
	}
	svc := newTestNotificationService(repo)

	updated, err := svc.MarkNotificationRead(context.Background(), uuid.NewString(), uuid.NewString())

	require.ErrorIs(t, err, expectedErr)
	assert.False(t, updated)
}

func TestMarkAllNotificationsReadSuccess(t *testing.T) {
	repo := &fakeNotificationRepo{
		markAllResult: 3,
	}
	svc := newTestNotificationService(repo)

	userID := uuid.New()

	count, err := svc.MarkAllNotificationsRead(context.Background(), userID.String())

	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.Equal(t, userID, repo.markAllRecipientID)
}

func TestMarkAllNotificationsReadInvalidUserID(t *testing.T) {
	repo := &fakeNotificationRepo{}
	svc := newTestNotificationService(repo)

	count, err := svc.MarkAllNotificationsRead(context.Background(), "bad-user-id")

	require.ErrorIs(t, err, ErrInvalidNotificationEvent)
	assert.Equal(t, int64(0), count)
}

func TestMarkAllNotificationsReadRepositoryError(t *testing.T) {
	expectedErr := errors.New("mark all failed")
	repo := &fakeNotificationRepo{
		markAllErr: expectedErr,
	}
	svc := newTestNotificationService(repo)

	count, err := svc.MarkAllNotificationsRead(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, int64(0), count)
}
