package repository

import (
	"context"

	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
)

type MediaRepository interface {
	Create(ctx context.Context, media *dbmodel.Media) error
}

type mediaRepository struct {
	db *gorm.DB
}

func NewMediaRepository(db *gorm.DB) MediaRepository {
	return &mediaRepository{db: db}
}

func (r *mediaRepository) Create(ctx context.Context, media *dbmodel.Media) error {
	return r.db.WithContext(ctx).Create(media).Error
}
