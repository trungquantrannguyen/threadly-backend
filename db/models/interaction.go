package model

import (
	"time"

	"github.com/google/uuid"
)

type Like struct {
	UserID uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	PostID uuid.UUID `gorm:"column:post_id;type:uuid;primaryKey"`

	CreatedAt time.Time `gorm:"column:created_at;not null"`

	User User `gorm:"foreignKey:UserID"`
	Post Post `gorm:"foreignKey:PostID"`
}

func (Like) TableName() string {
	return "likes"
}

type Bookmark struct {
	UserID uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	PostID uuid.UUID `gorm:"column:post_id;type:uuid;primaryKey"`

	CreatedAt time.Time `gorm:"column:created_at;not null"`

	User User `gorm:"foreignKey:UserID"`
	Post Post `gorm:"foreignKey:PostID"`
}

func (Bookmark) TableName() string {
	return "bookmarks"
}

type Repost struct {
	UserID uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	PostID uuid.UUID `gorm:"column:post_id;type:uuid;primaryKey"`

	CreatedAt time.Time `gorm:"column:created_at;not null"`

	User User `gorm:"foreignKey:UserID"`
	Post Post `gorm:"foreignKey:PostID"`
}

func (Repost) TableName() string {
	return "reposts"
}
