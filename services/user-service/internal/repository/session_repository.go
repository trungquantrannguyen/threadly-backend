package repository

import (
	"context"
	"errors"
	"time"

	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
)

var ErrSessionNotFound = errors.New("Session not found")

type SessionRepository interface {
	Create(ctx context.Context, session *dbmodel.Session) error
	FindActiveByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*dbmodel.Session, error)
	RevokeByID(ctx context.Context, sessionID string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{
		db: db,
	}
}

func (r *sessionRepository) Create(ctx context.Context, session *dbmodel.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) FindActiveByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*dbmodel.Session, error) {
	var session dbmodel.Session

	err := r.db.WithContext(ctx).
		Where("refresh_token_hash = ?", refreshTokenHash).
		Where("revoked_at IS NULL").
		Where("expires_at > ?", time.Now()).First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
	}

	return &session, nil
}

func (r *sessionRepository) RevokeByID(ctx context.Context, sessionID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&dbmodel.Session{}).
		Where("id = ?", sessionID).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)

	if result.Error != nil {
		return ErrSessionNotFound
	}
	return nil
}

func (r *sessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	now := time.Now()

	return r.db.WithContext(ctx).
		Model(&dbmodel.Session{}).
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Update("revoked_at", now).Error
}

func (r *sessionRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&dbmodel.Session{}).Error
}
