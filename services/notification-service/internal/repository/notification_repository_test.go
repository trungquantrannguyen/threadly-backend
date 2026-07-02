package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newNotificationRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	_, err = sqlDB.Exec(`
		CREATE TABLE users (
			id text PRIMARY KEY,
			email text NOT NULL UNIQUE,
			username text NOT NULL UNIQUE,
			password_hash text NOT NULL,
			display_name text NOT NULL,
			bio text,
			avatar_url text,
			banner_url text,
			location text,
			website_url text,
			is_verified boolean NOT NULL DEFAULT false,
			follower_count integer NOT NULL DEFAULT 0,
			following_count integer NOT NULL DEFAULT 0,
			post_count integer NOT NULL DEFAULT 0,
			created_at datetime NOT NULL,
			updated_at datetime NOT NULL,
			deleted_at datetime,
			role text NOT NULL DEFAULT 'user'
		);

		CREATE TABLE notifications (
			id text PRIMARY KEY,
			event_id text NOT NULL UNIQUE,
			recipient_id text NOT NULL,
			actor_id text,
			type text NOT NULL,
			entity_type text NOT NULL,
			entity_id text,
			payload text NOT NULL,
			read_at datetime,
			created_at datetime NOT NULL
		);

		CREATE INDEX idx_notifications_recipient_id ON notifications(recipient_id);
		CREATE INDEX idx_notifications_actor_id ON notifications(actor_id);
	`)
	require.NoError(t, err)

	return db
}

func closeNotificationRepositoryTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func newNotificationTestUser(username string) dbmodel.User {
	now := time.Now().UTC()

	return dbmodel.User{
		ID:           uuid.New(),
		Email:        username + "@example.com",
		Username:     username,
		PasswordHash: "hashed-password",
		DisplayName:  "Test User",
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func newTestNotification(recipientID uuid.UUID, actorID *uuid.UUID, createdAt time.Time) dbmodel.Notification {
	entityID := uuid.New()

	return dbmodel.Notification{
		ID:          uuid.New(),
		EventID:     uuid.New(),
		RecipientID: recipientID,
		ActorID:     actorID,
		Type:        "post_liked",
		EntityType:  "post",
		EntityID:    &entityID,
		Payload:     datatypes.JSON([]byte(`{"message":"liked your post"}`)),
		CreatedAt:   createdAt,
	}
}

func TestNotificationRepositoryCreate(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	actorID := uuid.New()
	notification := newTestNotification(recipientID, &actorID, time.Now().UTC())

	err := repo.Create(context.Background(), &notification)

	require.NoError(t, err)

	var saved dbmodel.Notification
	require.NoError(t, db.First(&saved, "id = ?", notification.ID).Error)

	assert.Equal(t, notification.ID, saved.ID)
	assert.Equal(t, notification.EventID, saved.EventID)
	assert.Equal(t, recipientID, saved.RecipientID)
	require.NotNil(t, saved.ActorID)
	assert.Equal(t, actorID, *saved.ActorID)
	assert.Equal(t, "post_liked", saved.Type)
	assert.Equal(t, "post", saved.EntityType)
	assert.JSONEq(t, `{"message":"liked your post"}`, string(saved.Payload))
}

func TestNotificationRepositoryCreateDuplicateEventIDDoesNothing(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	eventID := uuid.New()

	first := newTestNotification(recipientID, nil, time.Now().UTC())
	first.EventID = eventID

	second := newTestNotification(recipientID, nil, time.Now().UTC())
	second.EventID = eventID

	require.NoError(t, repo.Create(context.Background(), &first))
	require.NoError(t, repo.Create(context.Background(), &second))

	var count int64
	require.NoError(t, db.Model(&dbmodel.Notification{}).
		Where("event_id = ?", eventID).
		Count(&count).Error)

	assert.Equal(t, int64(1), count)
}

func TestNotificationRepositoryFindByRecipientID(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	otherRecipientID := uuid.New()

	actor := newNotificationTestUser("actor")
	require.NoError(t, db.Create(&actor).Error)

	older := newTestNotification(recipientID, &actor.ID, time.Now().UTC().Add(-2*time.Hour))
	newer := newTestNotification(recipientID, &actor.ID, time.Now().UTC().Add(-1*time.Hour))
	other := newTestNotification(otherRecipientID, &actor.ID, time.Now().UTC())

	require.NoError(t, db.Create(&older).Error)
	require.NoError(t, db.Create(&newer).Error)
	require.NoError(t, db.Create(&other).Error)

	notifications, err := repo.FindByRecipientID(context.Background(), recipientID, 10)

	require.NoError(t, err)
	require.Len(t, notifications, 2)

	assert.Equal(t, newer.ID, notifications[0].ID)
	assert.Equal(t, older.ID, notifications[1].ID)

	require.NotNil(t, notifications[0].Actor)
	assert.Equal(t, actor.ID, notifications[0].Actor.ID)
	assert.Equal(t, "actor", notifications[0].Actor.Username)
}

func TestNotificationRepositoryFindByRecipientIDUsesLimit(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()

	first := newTestNotification(recipientID, nil, time.Now().UTC().Add(-3*time.Hour))
	second := newTestNotification(recipientID, nil, time.Now().UTC().Add(-2*time.Hour))
	third := newTestNotification(recipientID, nil, time.Now().UTC().Add(-1*time.Hour))

	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)
	require.NoError(t, db.Create(&third).Error)

	notifications, err := repo.FindByRecipientID(context.Background(), recipientID, 2)

	require.NoError(t, err)
	require.Len(t, notifications, 2)

	assert.Equal(t, third.ID, notifications[0].ID)
	assert.Equal(t, second.ID, notifications[1].ID)
}

func TestNotificationRepositoryCountUnreadByRecipientID(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	otherRecipientID := uuid.New()
	now := time.Now().UTC()

	unreadOne := newTestNotification(recipientID, nil, now.Add(-3*time.Hour))
	unreadTwo := newTestNotification(recipientID, nil, now.Add(-2*time.Hour))
	read := newTestNotification(recipientID, nil, now.Add(-1*time.Hour))
	readAt := now
	read.ReadAt = &readAt

	other := newTestNotification(otherRecipientID, nil, now)

	require.NoError(t, db.Create(&unreadOne).Error)
	require.NoError(t, db.Create(&unreadTwo).Error)
	require.NoError(t, db.Create(&read).Error)
	require.NoError(t, db.Create(&other).Error)

	count, err := repo.CountUnreadByRecipientID(context.Background(), recipientID)

	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestNotificationRepositoryMarkAsRead(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	notification := newTestNotification(recipientID, nil, time.Now().UTC())

	require.NoError(t, db.Create(&notification).Error)

	updated, err := repo.MarkAsRead(context.Background(), notification.ID, recipientID)

	require.NoError(t, err)
	assert.True(t, updated)

	var saved dbmodel.Notification
	require.NoError(t, db.First(&saved, "id = ?", notification.ID).Error)
	assert.NotNil(t, saved.ReadAt)
}

func TestNotificationRepositoryMarkAsReadReturnsFalseWhenAlreadyRead(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	readAt := time.Now().UTC()

	notification := newTestNotification(recipientID, nil, time.Now().UTC())
	notification.ReadAt = &readAt

	require.NoError(t, db.Create(&notification).Error)

	updated, err := repo.MarkAsRead(context.Background(), notification.ID, recipientID)

	require.NoError(t, err)
	assert.False(t, updated)
}

func TestNotificationRepositoryMarkAsReadReturnsFalseForWrongRecipient(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	wrongRecipientID := uuid.New()

	notification := newTestNotification(recipientID, nil, time.Now().UTC())

	require.NoError(t, db.Create(&notification).Error)

	updated, err := repo.MarkAsRead(context.Background(), notification.ID, wrongRecipientID)

	require.NoError(t, err)
	assert.False(t, updated)
}

func TestNotificationRepositoryMarkAllAsRead(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	recipientID := uuid.New()
	otherRecipientID := uuid.New()
	now := time.Now().UTC()

	unreadOne := newTestNotification(recipientID, nil, now.Add(-3*time.Hour))
	unreadTwo := newTestNotification(recipientID, nil, now.Add(-2*time.Hour))

	alreadyRead := newTestNotification(recipientID, nil, now.Add(-1*time.Hour))
	readAt := now
	alreadyRead.ReadAt = &readAt

	otherRecipientUnread := newTestNotification(otherRecipientID, nil, now)

	require.NoError(t, db.Create(&unreadOne).Error)
	require.NoError(t, db.Create(&unreadTwo).Error)
	require.NoError(t, db.Create(&alreadyRead).Error)
	require.NoError(t, db.Create(&otherRecipientUnread).Error)

	updatedCount, err := repo.MarkAllAsRead(context.Background(), recipientID)

	require.NoError(t, err)
	assert.Equal(t, int64(2), updatedCount)

	var unreadCount int64
	require.NoError(t, db.Model(&dbmodel.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", recipientID).
		Count(&unreadCount).Error)

	assert.Equal(t, int64(0), unreadCount)

	var otherUnreadCount int64
	require.NoError(t, db.Model(&dbmodel.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", otherRecipientID).
		Count(&otherUnreadCount).Error)

	assert.Equal(t, int64(1), otherUnreadCount)
}

func TestNotificationRepositoryCreateDatabaseError(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	notification := newTestNotification(uuid.New(), nil, time.Now().UTC())

	closeNotificationRepositoryTestDB(t, db)

	err := repo.Create(context.Background(), &notification)

	require.Error(t, err)
}

func TestNotificationRepositoryFindByRecipientIDDatabaseError(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	closeNotificationRepositoryTestDB(t, db)

	notifications, err := repo.FindByRecipientID(context.Background(), uuid.New(), 10)

	require.Error(t, err)
	assert.Nil(t, notifications)
}

func TestNotificationRepositoryCountUnreadByRecipientIDDatabaseError(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	closeNotificationRepositoryTestDB(t, db)

	count, err := repo.CountUnreadByRecipientID(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
}

func TestNotificationRepositoryMarkAsReadDatabaseError(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	closeNotificationRepositoryTestDB(t, db)

	updated, err := repo.MarkAsRead(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.False(t, updated)
}

func TestNotificationRepositoryMarkAllAsReadDatabaseError(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	closeNotificationRepositoryTestDB(t, db)

	updatedCount, err := repo.MarkAllAsRead(context.Background(), uuid.New())

	require.Error(t, err)
	assert.Equal(t, int64(0), updatedCount)
}

func TestNotificationRepositoryDatabaseErrorIsNotNil(t *testing.T) {
	db := newNotificationRepositoryTestDB(t)
	repo := NewNotificationRepository(db)

	closeNotificationRepositoryTestDB(t, db)

	_, err := repo.CountUnreadByRecipientID(context.Background(), uuid.New())

	require.Error(t, err)
	assert.False(t, errors.Is(err, gorm.ErrRecordNotFound))
}
