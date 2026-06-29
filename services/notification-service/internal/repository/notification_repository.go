package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *dbmodel.Notification) error
	FindByRecipientID(ctx context.Context, recipientID uuid.UUID, limit int) ([]dbmodel.Notification, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID, recipientID uuid.UUID) (bool, error)
	MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) (int64, error)
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *dbmodel.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

func (r *notificationRepository) FindByRecipientID(ctx context.Context, recipientID uuid.UUID, limit int) ([]dbmodel.Notification, error) {
	var notifications []dbmodel.Notification

	err := r.db.WithContext(ctx).
		Where("recipient_id = ?", recipientID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error

	return notifications, err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, notificationID uuid.UUID, recipientID uuid.UUID) (bool, error) {
	now := time.Now().UTC()

	result := r.db.WithContext(ctx).
		Model(&dbmodel.Notification{}).
		Where("id = ? AND recipient_id = ? AND read_at IS NULL", notificationID, recipientID).
		Update("read_at", now)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	now := time.Now().UTC()

	result := r.db.WithContext(ctx).
		Model(&dbmodel.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", recipientID).
		Update("read_at", now)

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
