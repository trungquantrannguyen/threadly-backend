package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	AuthorID uuid.UUID `gorm:"column:author_id;type:uuid;not null;index"`

	// NULL = normal/root post
	// NOT NULL = reply to another post
	ReplyToPostID *uuid.UUID `gorm:"column:reply_to_post_id;type:uuid;index"`

	Content    string `gorm:"column:content;type:varchar(280);not null"`
	Visibility string `gorm:"column:visibility;type:varchar(20);not null;default:public"`

	LikeCount     int `gorm:"column:like_count;not null;default:0"`
	ReplyCount    int `gorm:"column:reply_count;not null;default:0"`
	RepostCount   int `gorm:"column:repost_count;not null;default:0"`
	BookmarkCount int `gorm:"column:bookmark_count;not null;default:0"`

	CreatedAt time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

	Author User `gorm:"foreignKey:AuthorID"`

	ReplyToPost *Post  `gorm:"foreignKey:ReplyToPostID"`
	Replies     []Post `gorm:"foreignKey:ReplyToPostID"`
}

func (Post) TableName() string {
	return "posts"
}
