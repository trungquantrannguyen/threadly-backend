package model

import (
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	UploaderID uuid.UUID  `gorm:"column:uploader_id;type:uuid;not null;index"`
	PostID     *uuid.UUID `gorm:"column:post_id;type:uuid;index"`

	URL        string `gorm:"column:url;type:text;not null"`
	StorageKey string `gorm:"column:storage_key;type:text;uniqueIndex;not null"`
	MimeType   string `gorm:"column:mime_type;type:varchar(100);not null"`
	SizeBytes  int64  `gorm:"column:size_bytes;not null"`
	Width      *int   `gorm:"column:width"`
	Height     *int   `gorm:"column:height"`
	Status     string `gorm:"column:status;type:varchar(20);not null;default:uploaded"`

	CreatedAt time.Time `gorm:"column:created_at;not null"`

	Uploader User  `gorm:"foreignKey:UploaderID"`
	Post     *Post `gorm:"foreignKey:PostID"`
}

func (Media) TableName() string {
	return "media"
}
